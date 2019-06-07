package resourcecreator_test

import (
	"fmt"
	"github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestIstio(t *testing.T) {
	t.Run("no service role resource created and no error when no access policy rules is defined and allow all is false", func(t *testing.T) {
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
		app.Spec.AccessPolicy.Ingress.AllowAll = true
		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{"a", ""}}

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.Error(t, err)
		assert.Nil(t, serviceRole)

		serviceRoleBinding, err := resourcecreator.ServiceRoleBinding(app)

		assert.Error(t, err)
		assert.Nil(t, serviceRoleBinding)
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

	t.Run("access policy with no specified namespace creates access rule with app namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Ingress.AllowAll = false


		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{"a", ""}}

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRole)
		assert.Equal(t, []*v1alpha1.AccessRule([]*v1alpha1.AccessRule{{[]string{ fmt.Sprintf("%s.%s.svc.cluster.local", "a", app.Namespace)}, []string{"*"}}}), serviceRole.Spec.Rules)
	})

	t.Run("access policy with specified namespace creates access rule with specified namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Ingress.AllowAll = false


		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{"a", "namespace"}}

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRole)
		assert.Equal(t, []*v1alpha1.AccessRule([]*v1alpha1.AccessRule{{[]string{ fmt.Sprintf("%s.%s.svc.cluster.local", "a", "namespace")}, []string{"*"}}}), serviceRole.Spec.Rules)
	})


}
