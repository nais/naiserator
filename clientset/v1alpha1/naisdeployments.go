package v1alpha1

import (
    "github.com/jhrv/operator/tutorial/api/types/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/watch"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
)

type NaisDeploymentInterface interface {
    List(opts metav1.ListOptions) (*v1alpha1.NaisDeploymentList, error)
    Get(name string, options metav1.GetOptions) (*v1alpha1.NaisDeployment, error)
    Create(*v1alpha1.NaisDeployment) (*v1alpha1.NaisDeployment, error)
    Watch(opts metav1.ListOptions) (watch.Interface, error)
    // ...
}

type naisDeploymentClient struct {
    restClient rest.Interface
    ns         string
}

func (c *naisDeploymentClient) List(opts metav1.ListOptions) (*v1alpha1.NaisDeploymentList, error) {
    result := v1alpha1.NaisDeploymentList{}
    err := c.restClient.
        Get().
        Namespace(c.ns).
        Resource("naisdeployments").
        VersionedParams(&opts, scheme.ParameterCodec).
        Do().
        Into(&result)

    return &result, err
}

func (c *naisDeploymentClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.NaisDeployment, error) {
    result := v1alpha1.NaisDeployment{}
    err := c.restClient.
        Get().
        Namespace(c.ns).
        Resource("naisdeployments").
        Name(name).
        VersionedParams(&opts, scheme.ParameterCodec).
        Do().
        Into(&result)

    return &result, err
}

func (c *naisDeploymentClient) Create(naisDeployment *v1alpha1.NaisDeployment) (*v1alpha1.NaisDeployment, error) {
    result := v1alpha1.NaisDeployment{}
    err := c.restClient.
        Post().
        Namespace(c.ns).
        Resource("naisdeployments").
        Body(naisDeployment).
        Do().
        Into(&result)

    return &result, err
}

func (c *naisDeploymentClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
    opts.Watch = true
    return c.restClient.
        Get().
        Namespace(c.ns).
        Resource("naisdeployments").
        VersionedParams(&opts, scheme.ParameterCodec).
        Watch()
}
