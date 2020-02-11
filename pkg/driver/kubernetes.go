package driver

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient ...
type KubeClient struct {
	Config  *rest.Config
	Client  *kubernetes.Clientset
	Dynamic dynamic.Interface
}

func (k *KubeClient) SetUserAgent(ua string) {
	k.Config.UserAgent = ua
}

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
		Config:  restConfig,
		Client:  clientSet,
		Dynamic: dynamicIntf,
	}, nil
}
