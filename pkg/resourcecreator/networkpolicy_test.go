package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	networking "k8s.io/api/networking/v1"
)

const accessPolicyApp = "allowedAccessApp"

func TestNetworkPolicy(t *testing.T) {
	t.Run("allow all policy sets rules to empty object", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Egress.AllowAll = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app)

		assert.Empty(t, networkPolicy.Spec.Egress[0].To)
		assert.Empty(t, networkPolicy.Spec.Egress[0].Ports)

	})

	t.Run("default deny all sets rules to empty slice", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Egress.AllowAll = false
		app.Spec.AccessPolicy.Ingress.AllowAll = false
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app)

		assert.Equal(t, []networking.NetworkPolicyEgressRule{}, networkPolicy.Spec.Egress)
		assert.Equal(t, []networking.NetworkPolicyIngressRule{}, networkPolicy.Spec.Ingress)
	})

	t.Run("allowed app in egress rule sets network policy pod selector to allowed app", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Egress.Rules = append(app.Spec.AccessPolicy.Egress.Rules, nais.AccessPolicyGressRule{Application: accessPolicyApp})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app)

		matchLabels := map[string]string{
			"app": accessPolicyApp,
		}

		assert.Equal(t, matchLabels, networkPolicy.Spec.Egress[0].To[0].PodSelector.MatchLabels)
	})
}
