package driver

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// ObjectDriver ...
type ObjectDriver interface {
	// Apply creates or updates the specified object.
	Apply(unstructured.Unstructured)

	// Delete deleted the specified object.
	Delete( /* TypeMeta? */ )

	// Adopt tells the driver to take ownership of and to start tracking
	// the specified object. Any adopted objects will be included in a
	// DeleteAll operation.
	Adopt(unstructured.Unstructured)

	DeleteAll()
}

func NewObjectDriver(client *KubeClient) ObjectDriver {
	return nil
}

// HydrateObject unmarshals YAML data into a unstructured.Unstructured
// object, applying any defaults and expanding templates.
func HydrateObject(objData []byte) *unstructured.Unstructured {
	return nil // TODO(jpeach): return error
}
