package doc

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/jpeach/modden/pkg/utils"
	"github.com/open-policy-agent/opa/ast"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// FragmentTypeUnknown indicates this Fragment is unknown
	// and needs to be decoded.
	FragmentTypeUnknown = iota
	// FragmentTypeInvalid indicates that this Fragment could not be parsed
	// or contains syntax errors.
	FragmentTypeInvalid
	// FragmentTypeObject indicates this Fragment contains a Kubernetes Object.
	FragmentTypeObject
	// FragmentTypeRego indicates this Fragment contains Rego.
	FragmentTypeRego
)

var _ error = &InvalidFragmentErr{}

// InvalidFragmentErr is an error value returned to indicate to the
// caller what type of fragment was found to be invalid.
type InvalidFragmentErr struct {
	// Type is the fragment type that was expected at the point
	// the error happened.
	Type FragmentType
}

func (e *InvalidFragmentErr) Error() string {
	return fmt.Sprintf("invalid %s fragment", e.Type)
}

// AsRegoCompilationErr attempts to convert this error into a Rego
// compilation error.
func AsRegoCompilationErr(err error) ast.Errors {
	var astErrors ast.Errors

	if errors.As(err, &astErrors) {
		return astErrors
	}

	return nil
}

// FragmentType is the parsed content type for the Fragment.
type FragmentType int

func (t FragmentType) String() string {
	switch t {
	case FragmentTypeObject:
		return "Kubernetes"
	case FragmentTypeRego:
		return "Rego"
	case FragmentTypeInvalid:
		return "invalid"
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

func hasKindVersion(u *unstructured.Unstructured) bool {
	k := u.GetObjectKind().GroupVersionKind()
	return len(k.Version) > 0 && len(k.Kind) > 0
}

func decodeYAMLOrJSON(data []byte) (*unstructured.Unstructured, error) {
	buffer := bytes.NewReader(data)
	decoder := yaml.NewYAMLOrJSONDecoder(buffer, buffer.Len())

	into := map[string]interface{}{}
	if err := decoder.Decode(&into); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: into}, nil
}

func regoModuleCompile(m *ast.Module) error {
	c := ast.NewCompiler()

	if c.Compile(map[string]*ast.Module{"check": m}); c.Failed() {
		return c.Errors
	}

	return nil
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
func (f *Fragment) Decode() (FragmentType, error) {
	if u, err := decodeYAMLOrJSON(f.Bytes); err == nil {
		// It's only a valid object if it has a version & kind.
		if hasKindVersion(u) {
			f.Type = FragmentTypeObject
			f.object = u
			return f.Type, nil
		}

		return FragmentTypeInvalid,
			utils.ChainErrors(
				&InvalidFragmentErr{Type: FragmentTypeObject},
				fmt.Errorf("YAML fragment is not a Kubernetes object"),
			)
	}

	if m, err := decodeModule(f.Bytes); err == nil {
		// Rego will parse raw JSON and YAML, but in that case there
		// won't be a any rules in the module.
		if m.Rules != nil && len(m.Rules) > 0 {
			f.Type = FragmentTypeRego
			f.module = m

			err = regoModuleCompile(m)
			if err != nil {
				return FragmentTypeInvalid,
					utils.ChainErrors(
						&InvalidFragmentErr{Type: FragmentTypeRego},
						err,
					)
			}

			return f.Type, nil
		}

	}

	return FragmentTypeUnknown, nil
}
