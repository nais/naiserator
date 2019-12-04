package resourcecreator_test

import (
	"fmt"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	rbac_istio_io_v1alpha1 "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
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

		serviceRole := resourcecreator.ServiceRole(app)

		assert.Nil(t, serviceRole)
	})

	t.Run("access policy rules defined when allow all is false throws an error", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, ""}}

		serviceRole := resourcecreator.ServiceRole(app)

		assert.NotNil(t, serviceRole)

		serviceRoleBinding := resourcecreator.ServiceRoleBinding(app)

		assert.NotNil(t, serviceRoleBinding)
	})

	t.Run("access policy with no specified namespace creates access rule with app namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{"*", app.Namespace}}

		serviceRole := resourcecreator.ServiceRole(app)

		assert.NotNil(t, serviceRole)

		service := fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace)
		assert.Equal(t, service, serviceRole.Spec.Rules[0].Services[0])
		assert.Len(t, serviceRole.Spec.Rules, 1)
	})

	t.Run("access policy with specified namespace creates serviceRoleBinding with specified namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}}

		serviceRoleBinding := resourcecreator.ServiceRoleBinding(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRoleBinding)

		user := fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication)
		assert.Equal(t, user, serviceRoleBinding.Spec.Subjects[0].User)
		assert.Len(t, serviceRoleBinding.Spec.Subjects, 1)
	})

	t.Run("access policy without specified namespace creates serviceRoleBinding with application namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, ""}}

		serviceRoleBinding := resourcecreator.ServiceRoleBinding(app)

		assert.NoError(t, err)
		assert.NotNil(t, serviceRoleBinding)
		user := fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication)
		assert.Equal(t, user, serviceRoleBinding.Spec.Subjects[0].User)
		assert.Len(t, serviceRoleBinding.Spec.Subjects, 1)
	})

	t.Run("specifying ingresses allows traffic from istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{
			"https://gief.api.plz",
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		serviceRoleBinding := resourcecreator.ServiceRoleBinding(app)
		assert.NoError(t, err)
		assert.NotNil(t, serviceRoleBinding)
		subject := rbac_istio_io_v1alpha1.Subject{
			User: "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account",
		}

		assert.Contains(t, serviceRoleBinding.Spec.Subjects, &subject)
	})

	t.Run("service role and service role binding created, with matching naming, when prometheus is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Prometheus.Path = "/metrics"
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		serviceRoleBindingPrometheus := resourcecreator.ServiceRoleBindingPrometheus(app)
		assert.NotNil(t, serviceRoleBindingPrometheus)

		assert.Equal(t, serviceRoleBindingPrometheus.ObjectMeta.Name, serviceRoleBindingPrometheus.ObjectMeta.Name)
	})

}
