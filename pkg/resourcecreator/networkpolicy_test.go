package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	networking "k8s.io/api/networking/v1"
)

const accessPolicyApp = "allowedAccessApp"
var defaultIps = []string{"12.0.0.0/12", "123.0.0.0/12"}

func TestNetworkPolicy(t *testing.T) {

	t.Run("default deny all sets app rules to empty slice", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)

		assert.Len(t, networkPolicy.Spec.Egress, 1)
		assert.Equal(t, []networking.NetworkPolicyIngressRule{}, networkPolicy.Spec.Ingress)
	})

	t.Run("allowed app in egress rule sets network policy pod selector to allowed app", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais.AccessPolicyRule{Application: accessPolicyApp})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)

		matchLabels := map[string]string{
			"app": accessPolicyApp,
		}

		assert.Equal(t, matchLabels, networkPolicy.Spec.Egress[1].To[0].PodSelector.MatchLabels)
	})

	t.Run("allowed app in egress rule sets egress app rules and default rules", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais.AccessPolicyRule{Application: accessPolicyApp})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)

		assert.Len(t, networkPolicy.Spec.Egress, 2)
	})

	t.Run("specifying ingresses allows traffic from istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{
			"https://gief.api.plz",
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)
		assert.NotNil(t, networkPolicy)
		assert.Len(t, networkPolicy.Spec.Ingress[0].From, 1)

		podMatch := map[string]string{"istio": "ingressgateway"}
		namespaceMatch := map[string]string{"name": "istio-system"}

		assert.Equal(t, podMatch, networkPolicy.Spec.Ingress[0].From[0].PodSelector.MatchLabels)
		assert.Equal(t, namespaceMatch, networkPolicy.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels)
	})

	t.Run("specifying ingresses when all traffic is allowed still creates an explicit rule for istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{
			"https://gief.api.plz",
		}
		app.Spec.AccessPolicy.Inbound.Rules = append(app.Spec.AccessPolicy.Inbound.Rules, nais.AccessPolicyRule{Application: "*"})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)
		assert.NotNil(t, networkPolicy)
		assert.Len(t, networkPolicy.Spec.Ingress, 2)
		assert.Len(t, networkPolicy.Spec.Ingress[0].From, 1)
		assert.Len(t, networkPolicy.Spec.Ingress[1].From, 1)

		podMatch := map[string]string{"istio": "ingressgateway"}
		namespaceMatch := map[string]string{"name": "istio-system"}

		assert.Equal(t, podMatch, networkPolicy.Spec.Ingress[1].From[0].PodSelector.MatchLabels)
		assert.Equal(t, namespaceMatch, networkPolicy.Spec.Ingress[1].From[0].NamespaceSelector.MatchLabels)
	})
	t.Run("all traffic inside namespace sets from rule to to empty podspec", func(t *testing.T) {
		app := fixtures.MinimalApplication()

		app.Spec.AccessPolicy.Inbound.Rules = append(app.Spec.AccessPolicy.Inbound.Rules, nais.AccessPolicyRule{Application: "*"})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)
		assert.NotNil(t, networkPolicy)

		yamlres, err := yaml.Marshal(networkPolicy)
		assert.NotNil(t, yamlres)
		assert.Empty(t, networkPolicy.Spec.Ingress[0].From[0].PodSelector)
	})

	t.Run("default network policy rule contains egress rules", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIps)
		assert.NotNil(t, networkPolicy)
		assert.Len(t, networkPolicy.Spec.Ingress, 0)
		assert.Len(t, networkPolicy.Spec.Egress, 1)
		assert.Len(t, networkPolicy.Spec.Egress[0].To, 4)

		podMatch := map[string]string{"istio": "pilot"}
		namespaceMatch := map[string]string{"name": "istio-system"}

		assert.Equal(t, podMatch, networkPolicy.Spec.Egress[0].To[0].PodSelector.MatchLabels)
		assert.Equal(t, namespaceMatch, networkPolicy.Spec.Egress[0].To[0].NamespaceSelector.MatchLabels)
		assert.Equal(t, defaultIps, networkPolicy.Spec.Egress[0].To[3].IPBlock.Except)
		assert.Equal(t, resourcecreator.NetworkPolicyDefaultEgressAllowIPBlock, networkPolicy.Spec.Egress[0].To[3].IPBlock.CIDR)
	})
}
