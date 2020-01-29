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

	t.Run("auth policy", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.NotNil(t, authorizationPolicy)
	})

	t.Run("auth policy no namespace", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, ""}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.NotNil(t, authorizationPolicy)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", app.Namespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
	})

	t.Run("auth policy for app with ingress", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"fjas"}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.NotNil(t, authorizationPolicy)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", resourcecreator.IstioNamespace, resourcecreator.IstioIngressGatewayServiceAccount), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
		assert.Len(t, authorizationPolicy.Spec.Rules, 2)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From, 1)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From[0].Source.Principals, 1)
		assert.Nil(t, authorizationPolicy.Spec.Rules[0].From[0].Source.Namespaces)
	})
	t.Run("auth policy for app with ingress and policy", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"fjas"}
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}, {otherApplication2, otherNamespace2}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.NotNil(t, authorizationPolicy)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", resourcecreator.IstioNamespace, resourcecreator.IstioIngressGatewayServiceAccount), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[1].Source.Principals[0])
		assert.Len(t, authorizationPolicy.Spec.Rules, 2)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From, 3)
	})
	t.Run("auth policy for app with multiple ingresses and policy", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"fjas", "fjos", "fis"}
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}, {otherApplication2, otherNamespace2}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.NotNil(t, authorizationPolicy)
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", resourcecreator.IstioNamespace, resourcecreator.IstioIngressGatewayServiceAccount), authorizationPolicy.Spec.Rules[0].From[0].Source.Principals[0])
		assert.Equal(t, fmt.Sprintf("cluster.local/ns/%s/sa/%s", otherNamespace, otherApplication), authorizationPolicy.Spec.Rules[0].From[1].Source.Principals[0])
		assert.Len(t, authorizationPolicy.Spec.Rules, 2)
		assert.Len(t, authorizationPolicy.Spec.Rules[0].From, 3)
	})
}
