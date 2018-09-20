package v1alpha1

import (
	"github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type NaisV1Alpha1Interface interface {
	Applications(namespace string) ApplicationInterface
}

type NaisV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*NaisV1Alpha1Client, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: v1alpha1.GroupName, Version: v1alpha1.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &NaisV1Alpha1Client{restClient: client}, nil
}

func (c *NaisV1Alpha1Client) Applications(namespace string) ApplicationInterface {
	return &applicationClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}
