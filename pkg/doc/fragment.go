package doc

import (
	"bytes"
	"io"

	"github.com/open-policy-agent/opa/ast"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// FragmentTypeUnknown indicates this Fragment is unknown
	// and needs to be decoded.
	FragmentTypeUnknown = iota
	// FragmentTypeObject indicates this Fragment contains a Kubernetes Object.
	FragmentTypeObject
	// FragmentTypeRego indicates this Fragment contains Rego.
	FragmentTypeRego
)

// FragmentType is the parsed content type for the Fragment.
type FragmentType int

func (t FragmentType) String() string {
	switch t {
	case FragmentTypeObject:
		return "Kubernetes"
	case FragmentTypeRego:
		return "Rego"
	default:
		return "unknown"
	}
}

// Fragment is a parseable portion of a Document.
type Fragment struct {
	Bytes []byte
	Type  FragmentType

	object *unstructured.Unstructured
	module *ast.Module
}

// Object returns the Kubernetes object if there is one.
func (f *Fragment) Object() *unstructured.Unstructured {
	switch f.Type {
	case FragmentTypeObject:
		return f.object
	default:
		return nil
	}
}

// Rego returns the Rego module if there is one.
func (f *Fragment) Rego() *ast.Module {
	switch f.Type {
	case FragmentTypeRego:
		return f.module
	default:
		return nil
	}
}

// TODO(jpeach): store a line number for the start of the fragment.

func decodeYAMLOrJSON(data []byte) (*unstructured.Unstructured, error) {
	buffer := bytes.NewReader(data)
	decoder := yaml.NewYAMLOrJSONDecoder(buffer, buffer.Len())

	into := map[string]interface{}{}
	err := decoder.Decode(&into)
	return &unstructured.Unstructured{Object: into}, err
}

func hasKindVersion(u *unstructured.Unstructured) bool {
	k := u.GetObjectKind().GroupVersionKind()
	return len(k.Version) > 0 && len(k.Kind) > 0
}

func decodeModule(data []byte) (*ast.Module, error) {
	// Rego requires a package name to generate any Rules.
	// For now, we force a "main" package since fragments
	// are anonymous.
	mod := "package main\n" + string(data)

	m, err := ast.ParseModule("main", mod)
	if err != nil {
		return nil, err
	}

	// ParseModule can return nil with no error (empty module).
	if m == nil {
		return nil, io.EOF
	}

	return m, nil
}

// Decode attempts to parse the Fragment.
func (f *Fragment) Decode() FragmentType {
	if u, err := decodeYAMLOrJSON(f.Bytes); err == nil {
		// It's only a valid object if it has a version & kind.
		if hasKindVersion(u) {
			f.Type = FragmentTypeObject
			f.object = u
			return f.Type
		}
	}

	if m, err := decodeModule(f.Bytes); err == nil {
		// Rego will parse raw JSON and YAML, but in that case there
		// won't be a any rules in the
		if m.Rules != nil && len(m.Rules) > 0 {
			f.Type = FragmentTypeRego
			f.module = m
			return f.Type
		}
	}

	return f.Type
}
