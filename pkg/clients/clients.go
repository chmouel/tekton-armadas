package clients

import (
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	tektonvclientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// ConnectMaxWaitTime most programming languages  do not have a timeout, but c# does a default
	// of 100 seconds so using that value.
	ConnectMaxWaitTime = 100 * time.Second
	RequestMaxWaitTime = 100 * time.Second
)

type Clients struct {
	Tekton            tektonvclientsetversioned.Interface
	Kube              kubernetes.Interface
	HTTP              http.Client
	ClientInitialized bool
}

func (c *Clients) kubeConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Parsing kubeconfig failed")
	}
	return config, nil
}

// Set kube client based on config.
func (c *Clients) kubeClient(config *rest.Config) (kubernetes.Interface, error) {
	k8scs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create k8s client from config")
	}

	return k8scs, nil
}

func (c *Clients) tektonClient(config *rest.Config) (tektonvclientsetversioned.Interface, error) {
	cs, err := tektonvclientsetversioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

func NewClients() (*Clients, error) {
	c := &Clients{}

	if c.ClientInitialized {
		return nil, nil
	}

	c.HTTP = http.Client{
		Timeout: RequestMaxWaitTime,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: ConnectMaxWaitTime,
			}).DialContext,
		},
	}

	config, err := c.kubeConfig()
	if err != nil {
		return nil, err
	}
	config.QPS = 50
	config.Burst = 50

	c.Kube, err = c.kubeClient(config)
	if err != nil {
		return nil, err
	}

	c.Tekton, err = c.tektonClient(config)
	if err != nil {
		return nil, err
	}

	c.ClientInitialized = true
	return c, nil
}
