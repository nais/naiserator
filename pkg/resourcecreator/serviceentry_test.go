package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestServiceEntry(t *testing.T) {

	t.Run("serviceentry not created if external hosts array omitted", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		serviceEntry := resourcecreator.ServiceEntry(app)
		assert.Nil(t, serviceEntry)
	})

	t.Run("serviceentry created according to spec", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Outbound.External = []nais.AccessPolicyExternalRule{
			{
				Host: "first.host",
			},
			{
				Host: "second.host",
			},
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		serviceEntry := resourcecreator.ServiceEntry(app)
		assert.NotNil(t, serviceEntry)

		assert.Equal(t, app.Name, serviceEntry.Name)
		assert.Equal(t, app.Namespace, serviceEntry.Namespace)

		assert.Equal(t, []string{"first.host", "second.host"}, serviceEntry.Spec.Hosts)
		assert.Equal(t, "MESH_EXTERNAL", serviceEntry.Spec.Location)
		assert.Equal(t, "DNS", serviceEntry.Spec.Resolution)

		assert.Len(t, serviceEntry.Spec.Ports, 1)
		assert.Equal(t, "HTTPS", serviceEntry.Spec.Ports[0].Protocol)
		assert.Equal(t, "https", serviceEntry.Spec.Ports[0].Name)
		assert.Equal(t, uint32(443), serviceEntry.Spec.Ports[0].Number)
	})
}
