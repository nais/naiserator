package ingress_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/ingress"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestIngress(t *testing.T) {
	var resourceOptions resource.Options
	resourceOptions.Linkerd = false

	t.Run("invalid ingress URLs are rejected", func(t *testing.T) {
		for _, i := range []nais.Ingress{"crap", "htp:/foo", "http://valid.fqdn/foo", "ftp://test"} {
			app := fixtures.MinimalApplication()
			app.Spec.Ingresses = []nais.Ingress{i}
			ast := resource.NewAst()
			err := nais.ApplyDefaults(app)
			assert.NoError(t, err)

			err = ingress.Create(app, ast, resourceOptions, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)

			assert.NotNil(t, err)
			assert.Len(t, ast.Operations, 0)
		}
	})
}
