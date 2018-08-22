package v1alpha1

import (
    "github.com/jhrv/operator/tutorial/api/types/v1alpha1"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/apimachinery/pkg/runtime/serializer"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
)

type NaisV1Alpha1Interface interface {
    NaisDeployments(namespace string) NaisDeploymentInterface
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

func (c *NaisV1Alpha1Client) NaisDeployments(namespace string) NaisDeploymentInterface {
    return &naisDeploymentClient{
        restClient: c.restClient,
        ns: namespace,
    }
}
