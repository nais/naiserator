// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"time"

	v1 "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	scheme "github.com/nais/naiserator/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// JwkersGetter has a method to return a JwkerInterface.
// A group's client should implement this interface.
type JwkersGetter interface {
	Jwkers(namespace string) JwkerInterface
}

// JwkerInterface has methods to work with Jwker resources.
type JwkerInterface interface {
	Create(*v1.Jwker) (*v1.Jwker, error)
	Update(*v1.Jwker) (*v1.Jwker, error)
	UpdateStatus(*v1.Jwker) (*v1.Jwker, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.Jwker, error)
	List(opts metav1.ListOptions) (*v1.JwkerList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Jwker, err error)
	JwkerExpansion
}

// jwkers implements JwkerInterface
type jwkers struct {
	client rest.Interface
	ns     string
}

// newJwkers returns a Jwkers
func newJwkers(c *NaiseratorV1Client, namespace string) *jwkers {
	return &jwkers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the jwker, and returns the corresponding jwker object, and an error if there is any.
func (c *jwkers) Get(name string, options metav1.GetOptions) (result *v1.Jwker, err error) {
	result = &v1.Jwker{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("jwkers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Jwkers that match those selectors.
func (c *jwkers) List(opts metav1.ListOptions) (result *v1.JwkerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.JwkerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("jwkers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested jwkers.
func (c *jwkers) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("jwkers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a jwker and creates it.  Returns the server's representation of the jwker, and an error, if there is any.
func (c *jwkers) Create(jwker *v1.Jwker) (result *v1.Jwker, err error) {
	result = &v1.Jwker{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("jwkers").
		Body(jwker).
		Do().
		Into(result)
	return
}

// Update takes the representation of a jwker and updates it. Returns the server's representation of the jwker, and an error, if there is any.
func (c *jwkers) Update(jwker *v1.Jwker) (result *v1.Jwker, err error) {
	result = &v1.Jwker{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("jwkers").
		Name(jwker.Name).
		Body(jwker).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *jwkers) UpdateStatus(jwker *v1.Jwker) (result *v1.Jwker, err error) {
	result = &v1.Jwker{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("jwkers").
		Name(jwker.Name).
		SubResource("status").
		Body(jwker).
		Do().
		Into(result)
	return
}

// Delete takes name of the jwker and deletes it. Returns an error if one occurs.
func (c *jwkers) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("jwkers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *jwkers) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("jwkers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched jwker.
func (c *jwkers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Jwker, err error) {
	result = &v1.Jwker{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("jwkers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}