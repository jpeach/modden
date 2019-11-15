package doc

import (
	"bytes"

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

// Decode attempts to parse the Fragment.
func (f Fragment) Decode() FragmentType {
	u, err := decodeYAMLOrJSON(f.Bytes)
	if err == nil {
		// It's only a valid object if it has a version & kind.
		if hasKindVersion(u) {
			f.Type = FragmentTypeObject
			f.object = u
			return f.Type
		}
	}

	return f.Type
}
