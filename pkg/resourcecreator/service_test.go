package resourcecreator_test

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetService(t *testing.T) {
	t.Run("Check if default port is used", func(t *testing.T) {
		svc := resourcecreator.Service(fixtures.Application())
		assert.Equal(t, nais.DefaultServicePort, int(svc.Spec.Ports[0].Port))
	})

	t.Run("check if correct port is used when set", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.Service.Port = 1337
		svc := resourcecreator.Service(app)
		assert.Equal(t, 1337, int(svc.Spec.Ports[0].Port))
	})
}
