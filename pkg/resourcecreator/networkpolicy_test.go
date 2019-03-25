package resourcecreator

import (
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	networking "k8s.io/api/networking/v1"
	"testing"
)

func TestNetworkPolicy(t *testing.T) {
	t.Run("Test that allow all policy sets rules to empty object", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.AccessPolicy.Egress.AllowAll = true
		networkPolicy := NetworkPolicy(app)

		assert.Empty(t, networkPolicy.Spec.Egress)
	})

	t.Run("Test that default deny all sets rules to empty list", func(t *testing.T) {
		app := fixtures.Application()
		networkPolicy := NetworkPolicy(app)

		assert.Equal(t, []networking.NetworkPolicyEgressRule{}, networkPolicy.Spec.Egress)
	})

}
