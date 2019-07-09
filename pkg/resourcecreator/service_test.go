package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetService(t *testing.T) {
	t.Run("Check if default port is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		svc := resourcecreator.Service(app)
		assert.Equal(t, nais.DefaultServicePort, int(svc.Spec.Ports[0].Port))
	})

	t.Run("check if correct port is used when set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Service.Port = 1337
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		svc := resourcecreator.Service(app)
		assert.Equal(t, 1337, int(svc.Spec.Ports[0].Port))
	})
}
