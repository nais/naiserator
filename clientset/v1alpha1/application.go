package v1alpha1

import (
    "github.com/nais/naiserator/api/types/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/watch"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
)

type ApplicationInterface interface {
    List(opts metav1.ListOptions) (*v1alpha1.ApplicationList, error)
    Get(name string, options metav1.GetOptions) (*v1alpha1.Application, error)
    Create(*v1alpha1.Application) (*v1alpha1.Application, error)
    Watch(opts metav1.ListOptions) (watch.Interface, error)
    // ...
}

type applicationClient struct {
    restClient rest.Interface
    ns         string
}

func (c *applicationClient) List(opts metav1.ListOptions) (*v1alpha1.ApplicationList, error) {
    result := v1alpha1.ApplicationList{}
    err := c.restClient.
        Get().
        Namespace(c.ns).
        Resource("applications").
        VersionedParams(&opts, scheme.ParameterCodec).
        Do().
        Into(&result)

    return &result, err
}

func (c *applicationClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.Application, error) {
    result := v1alpha1.Application{}
    err := c.restClient.
        Get().
        Namespace(c.ns).
        Resource("applications").
        Name(name).
        VersionedParams(&opts, scheme.ParameterCodec).
        Do().
        Into(&result)

    return &result, err
}

func (c *applicationClient) Create(application *v1alpha1.Application) (*v1alpha1.Application, error) {
    result := v1alpha1.Application{}
    err := c.restClient.
        Post().
        Namespace(c.ns).
        Resource("applications").
        Body(application).
        Do().
        Into(&result)

    return &result, err
}

func (c *applicationClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
    opts.Watch = true
    return c.restClient.
        Get().
        Namespace(c.ns).
        Resource("applications").
        VersionedParams(&opts, scheme.ParameterCodec).
        Watch()
}
