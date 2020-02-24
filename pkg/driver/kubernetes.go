package driver

import (
	"errors"
	"log"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient collects various Kubernetes client interfaces.
type KubeClient struct {
	Config    *rest.Config // XXX(jpeach): remove this, it's only needed for init
	Client    *kubernetes.Clientset
	Dynamic   dynamic.Interface
	Discovery discovery.CachedDiscoveryInterface
}

// SetUserAgent sets the HTTP User-Agent on the Client.
func (k *KubeClient) SetUserAgent(ua string) {
	// XXX(jpeach): user agent is captured at create time, so keeping the config here doesn't help ...
	k.Config.UserAgent = ua
}

// NamespaceExists tests whether the given namespace is present.
func (k *KubeClient) NamespaceExists(nsName string) (bool, error) {
	_, err := k.Client.CoreV1().Namespaces().Get(nsName, metav1.GetOptions{})
	switch {
	case err == nil:
		return true, nil
	case apierrors.IsNotFound(err):
		return false, nil
	default:
		return true, err
	}
}

func (k *KubeClient) findAPIResourceForKind(kind schema.GroupVersionKind) (metav1.APIResource, error) {
	resources, err := k.Discovery.ServerResourcesForGroupVersion(
		schema.GroupVersion{Group: kind.Group, Version: kind.Version}.String())
	if err != nil {
		return metav1.APIResource{}, err
	}

	// The listed resources will have empty Group and Version
	// fields, which means that they are the same as that of the
	// list. Parse the list's GroupVersion to populate the result.
	gv, err := schema.ParseGroupVersion(resources.GroupVersion)
	if err != nil {
		return metav1.APIResource{}, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == kind.Kind {
			if r.Group == "" {
				r.Group = gv.Group
			}

			if r.Version == "" {
				r.Version = gv.Version
			}

			return r, nil
		}
	}

	return metav1.APIResource{}, errors.New("no match for kind")
}

// KindIsNamespaced returns whether the given kind can be created within a namespace.
func (k *KubeClient) KindIsNamespaced(kind schema.GroupVersionKind) (bool, error) {
	res, err := k.findAPIResourceForKind(kind)
	if err != nil {
		return false, err
	}

	return res.Namespaced, nil
}

// ResourceForKind returns the schema.GroupVersionResource corresponding to kind.
func (k *KubeClient) ResourceForKind(kind schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	res, err := k.findAPIResourceForKind(kind)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	return schema.GroupVersionResource{
		Group:    res.Group,
		Version:  res.Version,
		Resource: res.Name,
	}, nil
}

// NewKubeClient returns a new set of Kubernetes client interfaces
// that are configured to use the default Kubernetes context.
func NewKubeClient() (*KubeClient, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	dynamicIntf, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Config:    restConfig,
		Client:    clientSet,
		Dynamic:   dynamicIntf,
		Discovery: memory.NewMemCacheClient(clientSet.Discovery()),
	}, nil
}

// NewNamespace returns a v1/Namespace object named by nsName and
// converted to an unstructured.Unstructured object.
func NewNamespace(nsName string) *unstructured.Unstructured {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}

	u := &unstructured.Unstructured{}

	if err := scheme.Scheme.Convert(ns, u, nil); err != nil {
		log.Fatalf("namespace conversion failed: %s", err)
	}

	return u
}

// ObjectReference uniquely identifies Kubernetes API object.
type ObjectReference struct {
	Name      string                  `json:"name"`
	Namespace string                  `json:"namespace"`
	Kind      schema.GroupVersionKind `json:"kind"`

	Meta struct {
		Group   string `json:"group"`
		Version string `json:"version"`
		Kind    string `json:"kind"`
	} `json:"meta"`
}

// FromUnstructured initializes an ObjectReference from a
// unstructured.Unstructured object.
func (o *ObjectReference) FromUnstructured(u *unstructured.Unstructured) *ObjectReference {

	o.Name = u.GetName()
	o.Namespace = u.GetNamespace()

	// We manually construct a GVK so that we can apply JSON
	// field labels to lowercase the names in the Rego data store.
	kind := u.GetObjectKind().GroupVersionKind()
	o.Meta.Group = kind.Group
	o.Meta.Version = kind.Version
	o.Meta.Kind = kind.Kind

	return o
}
