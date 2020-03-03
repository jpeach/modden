package driver

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jpeach/modden/pkg/must"
	"github.com/jpeach/modden/pkg/version"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
)

// DefaultResyncPeriod is the default informer resync interval.
const DefaultResyncPeriod = time.Minute * 5

// OperationResult describes the result of an attempt to apply a
// Kubernetes object update.
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
	Adopt(*unstructured.Unstructured) error

	DeleteAll()

	Watch(cache.ResourceEventHandler) func()

	Done()
}

// NewObjectDriver returns a new ObjectDriver.
func NewObjectDriver(client *KubeClient) ObjectDriver {
	selector := labels.SelectorFromSet(labels.Set{LabelManagedBy: version.Progname}).String()

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		client.Dynamic,
		DefaultResyncPeriod,
		metav1.NamespaceAll,
		func(o *metav1.ListOptions) {
			o.LabelSelector = selector
		},
	)

	o := &objectDriver{
		kube:            client,
		informerStopper: make(chan struct{}),
		informerFactory: factory,

		// watcherLock holds a lock over the watchers because
		// we need to ensure watcher add and remove operations
		// are serialized WRT event delivery.
		watcherLock: LockingResourceEventHandler{
			Next: &MuxingResourceEventHandler{},
		},

		objectPool:   make(map[types.UID]*unstructured.Unstructured),
		informerPool: make(map[schema.GroupVersionResource]informers.GenericInformer),
	}

	return o
}

var _ ObjectDriver = &objectDriver{}

type objectDriver struct {
	kube *KubeClient

	informerStopper chan struct{}
	informerFactory dynamicinformer.DynamicSharedInformerFactory

	watcherLock LockingResourceEventHandler

	informerPool map[schema.GroupVersionResource]informers.GenericInformer

	objectLock sync.Mutex
	objectPool map[types.UID]*unstructured.Unstructured
}

// Done resets the object driver.
func (o *objectDriver) Done() {
	// Tell any informers to shut down.
	close(o.informerStopper)

	// Hold the watcher lock while we clear the watchers.
	o.watcherLock.Lock.Lock()
	o.watcherLock.Next.(*MuxingResourceEventHandler).Clear()
	o.watcherLock.Lock.Unlock()

	// Hold the object lock while we cleat the object pool.
	o.objectLock.Lock()
	o.objectPool = make(map[types.UID]*unstructured.Unstructured)
	o.objectLock.Unlock()

	// There is no locking on the informer pool since driver
	// methods must not be called concurrently.
	o.informerPool = make(map[schema.GroupVersionResource]informers.GenericInformer)
}

func (o *objectDriver) Watch(e cache.ResourceEventHandler) func() {
	o.watcherLock.Lock.Lock()
	defer o.watcherLock.Lock.Unlock()

	which := o.watcherLock.Next.(*MuxingResourceEventHandler).Add(e)

	return func() {
		o.watcherLock.Lock.Lock()
		defer o.watcherLock.Lock.Unlock()

		o.watcherLock.Next.(*MuxingResourceEventHandler).Remove(which)
	}
}

func (o *objectDriver) Apply(obj *unstructured.Unstructured) (*OperationResult, error) {
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

	// If we don't already have an informer for this resource, start one now.
	if _, ok := o.informerPool[gvr]; !ok {
		genericInformer := o.informerFactory.ForResource(gvr)
		genericInformer.Informer().AddEventHandler(
			&WrappingResourceEventHandlerFuncs{
				Next: &o.watcherLock,
				AddFunc: func(obj interface{}) {
					o.objectLock.Lock()
					defer o.objectLock.Unlock()

					if u, ok := obj.(*unstructured.Unstructured); ok {
						o.updateAdoptedObject(u)
					}
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					o.objectLock.Lock()
					defer o.objectLock.Unlock()

					if u, ok := newObj.(*unstructured.Unstructured); ok {
						o.updateAdoptedObject(u)
					}
				},
				DeleteFunc: func(obj interface{}) {
					o.objectLock.Lock()
					defer o.objectLock.Unlock()

					if u, ok := obj.(*unstructured.Unstructured); ok {
						delete(o.objectPool, u.GetUID())
					}
				},
			})

		o.informerPool[gvr] = genericInformer

		go func() {
			genericInformer.Informer().Run(o.informerStopper)
		}()
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
		if err := o.Adopt(latest); err != nil {
			return nil, fmt.Errorf("failed to adopt %s %s/%s: %w",
				latest.GetKind(), latest.GetNamespace(), latest.GetName(), err)

		}

	default:
		var statusError *apierrors.StatusError
		if !errors.As(err, &statusError) {
			return nil, fmt.Errorf("failed to apply resource: %w", err)
		}

		result.Error = &statusError.ErrStatus
	}

	return &result, nil
}

func (o *objectDriver) Delete( /* TypeMeta? */ ) {
	panic("implement me")
}

func (o *objectDriver) updateAdoptedObject(obj *unstructured.Unstructured) {
	uid := obj.GetUID()

	// Update our adopted object only if it is from a newer generation.
	if prev, ok := o.objectPool[uid]; ok {
		if obj.GetGeneration() > prev.GetGeneration() {
			o.objectPool[uid] = obj.DeepCopy()
		}
	}
}

func (o *objectDriver) Adopt(obj *unstructured.Unstructured) error {
	o.objectLock.Lock()
	defer o.objectLock.Unlock()

	uid := obj.GetUID()

	// We can't adopt any object that hasn't come back from the
	// API server, since it isn't a legit object until then.
	if uid == "" {
		return errors.New("no object UID")
	}

	// Update our adopted object only if it is from a newer generation.
	if prev, ok := o.objectPool[uid]; ok {
		if obj.GetGeneration() > prev.GetGeneration() {
			o.objectPool[uid] = obj.DeepCopy()
		}
	} else {
		o.objectPool[uid] = obj.DeepCopy()
	}

	return nil
}

func (o *objectDriver) DeleteAll() {
	panic("implement me")
}
