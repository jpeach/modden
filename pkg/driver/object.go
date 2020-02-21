package driver

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ObjectDriver is a driver that is responsible for the lifecycle
// of Kubernetes API documents, expressed as unstructured.Unstructured
// objects.
type ObjectDriver interface {
	// Eval creates or updates the specified object.
	Apply(*unstructured.Unstructured) error

	// Delete deleted the specified object.
	Delete( /* TypeMeta? */ )

	// Adopt tells the driver to take ownership of and to start tracking
	// the specified object. Any adopted objects will be included in a
	// DeleteAll operation.
	Adopt(*unstructured.Unstructured)

	DeleteAll()
}

func NewObjectDriver(client *KubeClient) ObjectDriver {
	return &objectDriver{
		kube:       client,
		objectPool: make([]*unstructured.Unstructured, 0),
	}
}

var _ ObjectDriver = &objectDriver{}

type objectDriver struct {
	kube *KubeClient

	objectPool []*unstructured.Unstructured
}

func (o objectDriver) Apply(obj *unstructured.Unstructured) error {
	obj = obj.DeepCopy()
	gvk := obj.GetObjectKind().GroupVersionKind()

	isNamespaced, err := o.kube.KindIsNamespaced(gvk)
	if err != nil {
		return fmt.Errorf("failed check if resource kind is namespaced: %s", err)
	}

	gvr, err := o.kube.ResourceForKind(obj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return fmt.Errorf("failed to resolve resource for kind %s:%s: %s",
			obj.GetAPIVersion(), obj.GetKind(), err)
	}

	var result *unstructured.Unstructured

	if isNamespaced {
		if ns := obj.GetNamespace(); ns == "" {
			obj.SetNamespace("default")
		}

		result, err = o.kube.Dynamic.Resource(gvr).Namespace(obj.GetNamespace()).Create(obj, metav1.CreateOptions{})
	} else {
		result, err = o.kube.Dynamic.Resource(gvr).Create(obj, metav1.CreateOptions{})
	}

	if err != nil {
		// TODO(jpeach): return an error type that preserves
		// the underlying client-go error as well at attaches
		// the typemeta, gvr and kind information.
		return fmt.Errorf("failed to create resource: %s", err)
	}

	o.Adopt(result)
	return nil
}

func (o objectDriver) Delete( /* TypeMeta? */ ) {
	panic("implement me")
}

func (o objectDriver) Adopt(obj *unstructured.Unstructured) {
	o.objectPool = append(o.objectPool, obj)
}

func (o objectDriver) DeleteAll() {
	panic("implement me")
}
