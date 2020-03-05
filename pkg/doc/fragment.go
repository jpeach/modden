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
	// Force a package name that is unique to the fragment.
	moduleName := utils.RandomStringN(12)

	m, err := ast.ParseModule("main",
		fmt.Sprintf("package %s\n%s", moduleName, string(data)))
	if err != nil {
		// XXX(jpeach): if the parse fails, then we will think that
		// this fragment isn't Rego. But it could just be broken
		// Rego, in which case we ought to show a syntax error.
		return nil, err
	}

	// ParseModule can return nil with no error (empty module).
	if m == nil {
		return nil, io.EOF
	}

	return m, nil
}

// IsDecoded returns whether this fragment has been decoded to a known fragment type.
func (f *Fragment) IsDecoded() bool {
	switch f.Type {
	case FragmentTypeInvalid, FragmentTypeUnknown:
		return false
	default:
		return true
	}
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

	// At this point, we don't strictly know that this fragment
	// should decode to Rego. However, if we don't assume that,
	// then we can't know whether to propagate Rego syntax errors.
	// Since we do want to propagate errors so that users can debug
	// scripts, we have to assume this is meant to be Rego.

	m, err := decodeModule(f.Bytes)
	if err != nil {
		return FragmentTypeInvalid,
			utils.ChainErrors(
				&InvalidFragmentErr{Type: FragmentTypeRego}, err,
			)
	}

	// Rego will parse raw JSON and YAML, but in that
	// case there won't be a any rules in the module.
	if len(m.Rules) == 0 {
		return FragmentTypeUnknown, nil
	}

	// Compile the fragment so that we can
	// report syntax errors to the caller early.
	if err := regoModuleCompile(m); err != nil {
		return FragmentTypeInvalid,
			utils.ChainErrors(
				&InvalidFragmentErr{Type: FragmentTypeRego},
				err,
			)
	}

	f.Type = FragmentTypeRego
	f.module = m
	return f.Type, nil
}

// NewRegoFragment decodes the given data and returns a new Fragment
// of type FragmentTypeRego.
func NewRegoFragment(data []byte) (*Fragment, error) {
	frag := Fragment{Bytes: data}

	fragType, err := frag.Decode()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, AsRegoCompilationErr(err))
	}

	if fragType != FragmentTypeRego {
		return nil, fmt.Errorf("unexpected fragment type %q", fragType)
	}

	return &frag, nil
}
