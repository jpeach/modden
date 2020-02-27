package driver

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// LabelRunID is an annotation key to mark an object with
	// the unique ID of a test run.
	LabelRunID = "modden/run-id"

	// LabelVersion is ann annotation key to mark an object
	// with the version of the rest harness that created it.
	LabelVersion = "modden/version"
)

// SpecialOpsFilter is a yaml.Filter that extracts top-level YAML keys
// whose name begins with `$`. These keys denote special operations
// that test drivers need to interpolate.
type SpecialOpsFilter struct {
	Ops map[string]string
}

var _ yaml.Filter = &SpecialOpsFilter{}

// Filter runs the SpecialOpsFilter.
func (s *SpecialOpsFilter) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
	s.Ops = make(map[string]string)
	keep := make([]*yaml.Node, 0, len(rn.Content()))

	// Starting as index 0, we have alternate nodes for YAML
	// field names and YAML field values. A special ops field
	// is any field whose name begins with '$'.
	for i := 0; i < len(rn.Content()); i = yaml.IncrementFieldIndex(i) {
		key := rn.Content()[i]
		val := rn.Content()[i+1]

		// If the field name isn't a string, then who knows
		// what we should do. Skip it.
		if isStringNode(key) {
			if strings.HasPrefix(key.Value, "$") {
				s.Ops[key.Value] = val.Value
				// Return early so we filter out this key and value.
				continue
			}
		}

		keep = append(keep, key, val)
	}

	rn.YNode().Content = keep
	return rn, nil
}

func isStringNode(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode &&
		n.Tag == fieldmeta.String.Tag()
}

type MetaInjectionFilter struct {
	RunID     string
	ManagedBy string
}

var _ yaml.Filter = &MetaInjectionFilter{}

func (m *MetaInjectionFilter) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
	// First, inject the management label to the top object.
	if _, err := rn.Pipe(
		yaml.PathGetter{Create: yaml.MappingNode, Path: []string{"metadata", "labels"}},
		yaml.FieldSetter{Name: "app.kubernetes.io/managed-by", StringValue: m.ManagedBy},
	); err != nil {
		return nil, err
	}

	// Next, label the top level with the run ID.
	if _, err := rn.Pipe(
		yaml.PathGetter{Create: yaml.MappingNode, Path: []string{"metadata", "annotations"}},
		yaml.FieldSetter{Name: LabelRunID, StringValue: m.RunID},
	); err != nil {
		return nil, err
	}

	// Check whether this looks like an object that has a pod spec template.
	if c, err := rn.Pipe(
		yaml.PathGetter{Path: []string{"spec", "template", "spec", "containers"}},
	); c == nil || err != nil {
		return rn, err
	}

	// Since this object has a pod spec template, inject test metadata annotations into it.
	if _, err := rn.Pipe(
		yaml.PathGetter{Create: yaml.MappingNode, Path: []string{"spec", "template", "metadata", "annotations"}},
		yaml.FieldSetter{Name: LabelRunID, StringValue: m.RunID},
	); err != nil {
		return nil, err
	}

	return rn, nil

}

// yamlKindStr stringifies the yaml.Kind since kyaml doesn't do that for us.
// nolint:unused,deadcode
func yamlKindStr(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "Document"
	case yaml.SequenceNode:
		return "Sequence"
	case yaml.MappingNode:
		return "Mapping"
	case yaml.ScalarNode:
		return "Scalar"
	case yaml.AliasNode:
		return "Alias"
	default:
		return "huh?"
	}
}
