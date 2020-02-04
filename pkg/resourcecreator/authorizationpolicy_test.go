package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
)

func TestGetAuthorizationPolicy(t *testing.T) {
	otherApplication := "a"
	otherNamespace := "othernamespace"
	otherApplication2 := "b"
	otherNamespace2 := "othernamespace2"

	t.Run("auth policy with no ingresses or access policies", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.Len(t, authorizationPolicy.Spec.Rules, 0)
	})

	t.Run("auth policy no namespace or ingress", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, ""}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})

	t.Run("auth policy for app with ingress and no access policies", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"fjas"}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", resourcecreator.IstioNamespace, resourcecreator.IstioIngressGatewayServiceAccount), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})
	t.Run("auth policy for app with ingress and policy", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"fjas"}
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}, {otherApplication2, ""}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.Len(t, authorizationPolicy.Spec.Rules, 2)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication), authorizationPolicy.Spec.Rules[1].From[0].Source.Principals[0])
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication2), authorizationPolicy.Spec.Rules[1].From[0].Source.Principals[1])
	})
	t.Run("auth policy for app with access policy and no ingress", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From[0].Source.Principals, 1)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})
	t.Run("auth policy for app with multiple inbound", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}, {otherApplication2, otherNamespace2}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.Len(t, authorizationPolicy.Spec.Rules, 1)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From[0].Source.Principals, 2)
	})
}
