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

	otherApplication := "a"
	otherNamespace := "othernamespace"

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
		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{otherApplication, ""}}

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


		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{otherApplication, ""}}

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRole)
		assert.Equal(t, []*v1alpha1.AccessRule([]*v1alpha1.AccessRule{{[]string{ fmt.Sprintf("%s.%s.svc.cluster.local", otherApplication, app.Namespace)}, []string{"*"}}}), serviceRole.Spec.Rules)
	})

	t.Run("access policy with specified namespace creates access rule with specified namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Ingress.AllowAll = false

		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{otherApplication, otherNamespace}}

		serviceRole, err := resourcecreator.ServiceRole(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRole)
		assert.Equal(t, []*v1alpha1.AccessRule([]*v1alpha1.AccessRule{{[]string{ fmt.Sprintf("%s.%s.svc.cluster.local", otherApplication, otherNamespace)}, []string{"*"}}}), serviceRole.Spec.Rules)
	})


	t.Run("access policy with specified namespace creates serviceRoleBinding with specified namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{otherApplication, otherNamespace}}

		serviceRoleBinding, err := resourcecreator.ServiceRoleBinding(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRoleBinding)
		assert.Equal(t, []*v1alpha1.Subject([]*v1alpha1.Subject{{User: fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication)}}), serviceRoleBinding.Spec.Subjects)
	})

	t.Run("access policy without specified namespace creates serviceRoleBinding with application namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.AccessPolicy.Ingress.Rules = []nais.AccessPolicyGressRule{{otherApplication, ""}}

		serviceRoleBinding, err := resourcecreator.ServiceRoleBinding(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRoleBinding)
		assert.Equal(t, []*v1alpha1.Subject([]*v1alpha1.Subject{{User: fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication)}}), serviceRoleBinding.Spec.Subjects)
	})

}
