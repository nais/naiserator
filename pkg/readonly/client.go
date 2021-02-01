package readonly

import (
	"context"

	naiserator_scheme "github.com/nais/naiserator/pkg/naiserator/scheme"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ client.Client = &Client{}

// Return a copy of `c` with write privileges dropped.
func NewClient(c client.Client) client.Client {
	return &Client{
		client: c,
	}
}

type Client struct {
	client client.Client
}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	// log.Debugf("Read-only client: GET %s", naiserator_scheme.TypeName(obj))
	return c.client.Get(ctx, key, obj)
}

func (c *Client) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	// log.Debugf("Read-only client: LIST %s", naiserator_scheme.TypeName(list))
	return c.client.List(ctx, list, opts...)
}

func (c *Client) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	log.Debugf("Read-only client ignoring CREATE %s", naiserator_scheme.TypeName(obj))
	return nil
}

func (c *Client) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	log.Debugf("Read-only client ignoring DELETE %s", naiserator_scheme.TypeName(obj))
	return nil
}

func (c *Client) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	log.Debugf("Read-only client ignoring UPDATE %s", naiserator_scheme.TypeName(obj))
	return nil
}

func (c *Client) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	log.Debugf("Read-only client ignoring PATCH %s", naiserator_scheme.TypeName(obj))
	return nil
}

func (c *Client) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	log.Debugf("Read-only client ignoring DELETE ALL OF %s", naiserator_scheme.TypeName(obj))
	return nil
}

func (c *Client) Status() client.StatusWriter {
	return c
}
