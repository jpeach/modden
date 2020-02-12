package driver

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Environment interface {
	// UniqueID returns a unique identifier for this Environment instance.
	UniqueID() string

	// HydrateObject ...
	HydrateObject(objData []byte) (*unstructured.Unstructured, error)
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

// HydrateObject unmarshals YAML data into a unstructured.Unstructured
// object, applying any defaults and expanding templates.
func (e *environ) HydrateObject(objData []byte) (*unstructured.Unstructured, error) {
	// TODO(jpeach): before parsing YAML, apply Go template context.

	resource, err := yaml.Parse(string(objData))
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

	u := &unstructured.Unstructured{}

	_, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonBytes, nil, u)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %s", err)
	}

	// TODO(jpeach): Now that we are Unstructured, make any generic modifications.

	return u, nil
}
