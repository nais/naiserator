package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetAuthorizationPolicy(t *testing.T) {
	otherApplication := "a"
	otherNamespace := "othernamespace"
	otherApplication2 := "b"
	otherNamespace2 := "othernamespace2"
	resourceOptions := resourcecreator.NewResourceOptions()
	resourceOptions.GatewayMappings = []config.GatewayMapping{{DomainSuffix: ".test.no", GatewayName: "istio-system/gw-test"}}
	resourceOptions.ClusterName = "test-cluster"

	t.Run("auth policy with no ingresses or access policies", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Nil(t, authorizationPolicy)
	})

	t.Run("auth policy no namespace or ingress", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, "", ""}}
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})

	t.Run("auth policy for app with ingress and no access policies", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{"https://asd.test.no"}
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", "istio-system", "gw-test-service-account"), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", resourcecreator.IstioNamespace, resourcecreator.IstioIngressGatewayServiceAccount), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[1])
	})
	t.Run("auth policy for app with ingress and policy", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{"https://asd.test.no"}
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, ""}, {otherApplication2, "", ""}}
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Len(t, authorizationPolicy.Spec.Rules, 2)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication), authorizationPolicy.Spec.Rules[1].From[0].Source.Principals[0])
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication2), authorizationPolicy.Spec.Rules[1].From[0].Source.Principals[1])
	})
	t.Run("auth policy for app with access policy and no ingress", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, ""}}
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From[0].Source.Principals, 1)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})
	t.Run("auth policy for app with multiple inbound", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, ""}, {otherApplication2, otherNamespace2, ""}}
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From[0].Source.Principals, 2)
	})
	t.Run("auth policy for app with inbound access policy containing only non-local principals", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, "non-local"}}
		authorizationPolicy, err := resourcecreator.AuthorizationPolicy(app, resourceOptions)
		assert.NoError(t, err)
		assert.Nil(t, authorizationPolicy)
	})
}
