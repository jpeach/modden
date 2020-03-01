package driver

import (
	"errors"
	"fmt"

	"github.com/jpeach/modden/pkg/must"
	"k8s.io/client-go/kubernetes/scheme"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// OperationResult captures the result of applying a Kubernetes object.
type OperationResult struct {
	Error  *metav1.Status             `json:"error"`
	Latest *unstructured.Unstructured `json:"latest"`
	Target ObjectReference            `json:"target"`
}

// Succeeded returns true if the operation was successful.
func (o *OperationResult) Succeeded() bool {
	return o.Error == nil
}

// ObjectDriver is a driver that is responsible for the lifecycle
// of Kubernetes API documents, expressed as unstructured.Unstructured
// objects.
type ObjectDriver interface {
	// Eval creates or updates the specified object.
	Apply(*unstructured.Unstructured) (*OperationResult, error)

	// Delete deleted the specified object.
	Delete( /* TypeMeta? */ )

	// Adopt tells the driver to take ownership of and to start tracking
	// the specified object. Any adopted objects will be included in a
	// DeleteAll operation.
	Adopt(*unstructured.Unstructured)

	DeleteAll()
}

// NewObjectDriver returns a new ObjectDriver.
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

func (o objectDriver) Apply(obj *unstructured.Unstructured) (*OperationResult, error) {
	obj = obj.DeepCopy()
	gvk := obj.GetObjectKind().GroupVersionKind()

	isNamespaced, err := o.kube.KindIsNamespaced(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed check if resource kind is namespaced: %s", err)
	}

	gvr, err := o.kube.ResourceForKind(obj.GetObjectKind().GroupVersionKind())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve resource for kind %s:%s: %s",
			obj.GetAPIVersion(), obj.GetKind(), err)
	}

	if isNamespaced {
		if ns := obj.GetNamespace(); ns == "" {
			obj.SetNamespace("default")
		}
	}

	var latest *unstructured.Unstructured

	if isNamespaced {
		latest, err = o.kube.Dynamic.Resource(gvr).Namespace(obj.GetNamespace()).Create(obj, metav1.CreateOptions{})
	} else {
		latest, err = o.kube.Dynamic.Resource(gvr).Create(obj, metav1.CreateOptions{})
	}

	// If the create was against an object that already existed,
	// retry as an update.
	if apierrors.IsAlreadyExists(err) {
		name := obj.GetName()
		opt := metav1.PatchOptions{}
		ptype := types.MergePatchType
		data := must.Bytes(obj.MarshalJSON())

		// This is a hacky shortcut to emulate what kubectl
		// does in apply.Patcher. Since only built-in types
		// support strategic merge, we use the scheme check
		// to test whether this object is builtin or not.
		if _, err := scheme.Scheme.New(obj.GroupVersionKind()); err == nil {
			ptype = types.StrategicMergePatchType
		}

		if isNamespaced {
			latest, err = o.kube.Dynamic.Resource(gvr).Namespace(obj.GetNamespace()).Patch(name, ptype, data, opt)
		} else {
			latest, err = o.kube.Dynamic.Resource(gvr).Patch(name, ptype, data, opt)
		}
	}

	result := OperationResult{
		Error:  nil,
		Latest: obj,
		Target: *(&ObjectReference{}).FromUnstructured(obj),
	}

	switch err {
	case nil:
		result.Latest = latest
		o.Adopt(latest.DeepCopy())

	default:
		var statusError *apierrors.StatusError
		if !errors.As(err, &statusError) {
			return nil, fmt.Errorf("failed to apply resource: %w", err)
		}

		result.Error = &statusError.ErrStatus
	}

	return &result, nil
}

func (o objectDriver) Delete( /* TypeMeta? */ ) {
	panic("implement me")
}

func (o objectDriver) Adopt(obj *unstructured.Unstructured) {
	//TODO(jpeach): index the object pool by something?
	o.objectPool = append(o.objectPool, obj)
}

func (o objectDriver) DeleteAll() {
	panic("implement me")
}
