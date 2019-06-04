package resourcecreator

import (
	"testing"

	"github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	networking "k8s.io/api/networking/v1"
)

func TestNetworkPolicy(t *testing.T) {
	t.Run("Test that allow all policy sets rules to empty object", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.AccessPolicy.Egress.AllowAll = true
		networkPolicy := NetworkPolicy(app)

		assert.Empty(t, networkPolicy.Spec.Egress[0].To)
		assert.Empty(t, networkPolicy.Spec.Egress[0].Ports)

	})

	t.Run("Test that default deny all sets rules to empty slice", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.AccessPolicy.Egress.AllowAll = false
		app.Spec.AccessPolicy.Ingress.AllowAll = false
		networkPolicy := NetworkPolicy(app)

		assert.Equal(t, []networking.NetworkPolicyEgressRule{}, networkPolicy.Spec.Egress)
		assert.Equal(t, []networking.NetworkPolicyIngressRule{}, networkPolicy.Spec.Ingress)
	})

	t.Run("Allowed app in egress rule sets network policy pod selector to allowed app", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.AccessPolicy.Egress.Rules = append(app.Spec.AccessPolicy.Egress.Rules, v1alpha1.AccessPolicyGressRule{Application: fixtures.AccessPolicyApp})
		networkPolicy := NetworkPolicy(app)

		matchLabels := map[string]string{
			"app": fixtures.AccessPolicyApp,
		}

		assert.Equal(t, matchLabels, networkPolicy.Spec.Egress[0].To[0].PodSelector.MatchLabels)
	})
}
