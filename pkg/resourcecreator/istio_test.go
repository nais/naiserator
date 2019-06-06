package resourcecreator_test

import (
	"github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestIstio(t *testing.T) {
	t.Run("no service role resource created and no error with access policy rules is defined and allow all is false", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.NoError(t, err)
		assert.Nil(t, serviceRole)
	})

	t.Run("access policy rules defined when allow all is false throws an error", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{"a", ""}}

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.Error(t, err)
		assert.Nil(t, serviceRole)
	})

	t.Run("access policy with allow all returns servicerole with * as access rule", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Ingress.AllowAll = true

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRole)
		assert.Equal(t, []*v1alpha1.AccessRule([]*v1alpha1.AccessRule{{[]string{"*"}, []string{"*"}}}), serviceRole.Spec.Rules)
	})

}
