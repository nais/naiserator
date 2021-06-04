package service_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
)

func TestGetService(t *testing.T) {
	t.Run("Check if default values is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		err := nais.ApplyApplicationDefaults(app)
		assert.NoError(t, err)

		service.Create(app, ast, *app.Spec.Service)
		svc := ast.Operations[0].Resource.(*core.Service)
		port := svc.Spec.Ports[0]
		assert.Equal(t, nais.DefaultPortName, port.Name)
		assert.Equal(t, nais.DefaultServicePort, int(port.Port))
	})

	t.Run("check if correct value is used when set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.Service.Protocol = "redis"
		app.Spec.Service.Port = 1337
		err := nais.ApplyApplicationDefaults(app)
		assert.NoError(t, err)

		service.Create(app, ast, *app.Spec.Service)
		svc := ast.Operations[0].Resource.(*core.Service)
		port := svc.Spec.Ports[0]
		assert.Equal(t, "redis", port.Name)
		assert.Equal(t, 1337, int(port.Port))
	})
}
