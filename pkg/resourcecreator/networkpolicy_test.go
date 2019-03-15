package resourcecreator

import (
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	networking "k8s.io/api/networking/v1"
	"testing"
)

func TestNetworkPolicy(t *testing.T) {
	t.Run("Test that allow all removes all policy types", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.AccessPolicy.Egress.AllowAll = true
		app.Spec.AccessPolicy.Ingress.AllowAll = true
		networkPolicy := NetworkPolicy(app)

		assert.Empty(t, networkPolicy.Spec.PolicyTypes)
	})

	t.Run("If allow all egress, only ingress policy type in NetworkPolicy", func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.AccessPolicy.Egress.Rules = append(app.Spec.AccessPolicy.Egress.Rules, "example.com")
		networkPolicy := NetworkPolicy(app)

		assert.Len(t, networkPolicy.Spec.PolicyTypes, 1)
		assert.Equal(t, networking.PolicyTypeIngress, networkPolicy.Spec.PolicyTypes[0])
	})
}
