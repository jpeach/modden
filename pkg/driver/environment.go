package driver

import (
	"fmt"
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/filter"
	"github.com/jpeach/modden/pkg/version"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Environment holds metadata that describes the context of a test.
type Environment interface {
	// UniqueID returns a unique identifier for this Environment instance.
	UniqueID() string

	// HydrateObject ...
	HydrateObject(objData []byte) (*Object, error)
}

// NewEnvironment returns a new Environment.
func NewEnvironment() Environment {
	return &environ{
		uid: uuid.New().String(),
	}
}

var _ Environment = &environ{}

type environ struct {
	uid string
}

// UniqueID returns a unique identifier for this Environment instance.
func (e *environ) UniqueID() string {
	return e.uid
}

// ObjectOperationType desscribes the type of operation to apply
// to this object. This is derived from the "$apply" pseudo-field.
type ObjectOperationType string

const (
	// ObjectOperationDelete indicates this object should be deleted.
	ObjectOperationDelete = "delete"
	// ObjectOperationUpdate indicates this object should be
	// updated (i.e created or patched).
	ObjectOperationUpdate = "update"
)

// Object captures an Unstructured Kubernetes API object and its
// associated metadata.
//
// TODO(jpeach): this is a terrible name. Refactor this whole bizarre atrocity.
type Object struct {
	// Object is the object to apply.
	Object *unstructured.Unstructured

	// Check is a Rego check to run on the apply.
	Check *doc.Fragment

	// Delete specifies whether we are updating or deleting the object.
	Operation ObjectOperationType
}

// HydrateObject unmarshals YAML data into a unstructured.Unstructured
// object, applying any defaults and expanding templates.
func (e *environ) HydrateObject(objData []byte) (*Object, error) {
	// TODO(jpeach): before parsing YAML, apply Go template context.

	resource, err := yaml.Parse(string(objData))
	if err != nil {
		return nil, err
	}

	// Filter out any special operations.
	ops := filter.SpecialOpsFilter{}
	resource, err = resource.Pipe(&ops)
	if err != nil {
		return nil, err
	}

	// Inject test metadata.
	resource, err = resource.Pipe(
		&filter.MetaInjectionFilter{RunID: e.UniqueID(), ManagedBy: version.Progname})
	if err != nil {
		return nil, err
	}

	meta, err := resource.GetMeta()
	if err != nil {
		return nil, err
	}

	id := meta.GetIdentifier()
	if id.Namespace == "" {
		id.Namespace = "default"
	}

	// TODO(jpeach): Now apply kustomizations. If this is a
	// fragment rather than a whole object document, then we
	// need to expand it before parsing.

	jsonBytes, err := yamlutil.ToJSON([]byte(resource.MustString()))
	if err != nil {
		return nil, err
	}

	o := Object{
		Object:    &unstructured.Unstructured{},
		Operation: ObjectOperationUpdate,
	}

	// TODO(jpeach): Now that we are Unstructured, make any generic modifications.
	if what, ok := ops.Ops["$apply"]; ok {
		switch what {
		case "update":
			// This is the default.
			o.Operation = ObjectOperationUpdate
		case "delete":
			o.Operation = ObjectOperationDelete
		default:
			log.Printf("invalid object operation %q", what)
		}

	}

	if _, ok := ops.Ops["$check"]; ok {
		frag, err := doc.NewRegoFragment([]byte(ops.Ops["$check"]))
		if err != nil {
			// TODO(jpeach): send to test listener.
			panic(err)
		}

		o.Check = frag
	}

	_, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonBytes, nil, o.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %s", err)
	}

	return &o, nil
}
