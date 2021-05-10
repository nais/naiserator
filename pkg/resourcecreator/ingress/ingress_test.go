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
	var options resource.Options
	options.Linkerd = false

	t.Run("invalid ingress URLs are rejected", func(t *testing.T) {
		for _, i := range []nais.Ingress{"crap", "htp:/foo", "http://valid.fqdn/foo", "ftp://test"} {
			app := fixtures.MinimalApplication()
			app.Spec.Ingresses = []nais.Ingress{i}
			ops := resource.Operations{}
			err := nais.ApplyDefaults(app)
			assert.NoError(t, err)

			err = ingress.Create(app, options, &ops)

			assert.NotNil(t, err)
			assert.Len(t, ops, 0)
		}
	})
}
