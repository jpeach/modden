package driver

import (
	"fmt"
	"log"

	"github.com/jpeach/modden/pkg/doc"
	"github.com/jpeach/modden/pkg/version"

	"github.com/google/uuid"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Environment interface {
	// UniqueID returns a unique identifier for this Environment instance.
	UniqueID() string

	// HydrateObject ...
	HydrateObject(objData []byte) (*Object, error)
}

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
	Delete bool
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
	ops := SpecialOpsFilter{}
	resource, err = resource.Pipe(&ops)
	if err != nil {
		return nil, err
	}

	// Inject test metadata.
	resource, err = resource.Pipe(
		&MetaInjectionFilter{RunID: e.UniqueID(), ManagedBy: version.Progname})
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

	log.Printf("fragment contains %s:%s object %s/%s",
		id.APIVersion, id.Kind, id.Namespace, id.Name)

	// TODO(jpeach): Now apply kustomizations. If this is a
	// fragment rather than a whole object document, then we
	// need to expand it before parsing.

	jsonBytes, err := yamlutil.ToJSON([]byte(resource.MustString()))
	if err != nil {
		return nil, err
	}

	o := Object{
		Object: &unstructured.Unstructured{},
	}

	// TODO(jpeach): Now that we are Unstructured, make any generic modifications.
	if what, ok := ops.Ops["$apply"]; ok {
		switch what {
		case "update":
			// This is the default.
		case "delete":
			o.Delete = true
		case "patch":
		// TODO(jpeach): apply this as a structured merge patch.
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
