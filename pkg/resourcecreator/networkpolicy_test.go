package resourcecreator_test

import (
	"github.com/nais/naiserator/pkg/resourcecreator/resourceutils"
	"testing"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const accessPolicyApp = "allowedAccessApp"

var defaultIps = []string{"12.0.0.0/12", "123.0.0.0/12"}

func TestNetworkPolicy(t *testing.T) {

	defaultIpsOptions := resourceutils.NewResourceOptions()

	defaultIpsOptions.Istio = true
	defaultIpsOptions.AccessPolicyNotAllowedCIDRs = defaultIps

	t.Run("default deny all sets app rules to empty slice", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIpsOptions)

		assert.Len(t, networkPolicy.Spec.Egress, 1)

		testPolicy := make([]networking.NetworkPolicyIngressRule, 0)

		testPolicy = append(testPolicy, networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "prometheus",
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"name": "istio-system",
						},
					},
				},
			},
		})

		assert.Equal(t, testPolicy, networkPolicy.Spec.Ingress)
	})

	t.Run("allowed app in egress rule sets network policy pod selector to allowed app", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais_io_v1.AccessPolicyRule{Application: accessPolicyApp})
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIpsOptions)

		matchLabels := map[string]string{
			"app": accessPolicyApp,
		}

		assert.Equal(t, matchLabels, networkPolicy.Spec.Egress[1].To[0].PodSelector.MatchLabels)
	})

	t.Run("allowed app in egress rule sets egress app rules and default rules", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais_io_v1.AccessPolicyRule{Application: accessPolicyApp})
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIpsOptions)

		assert.Len(t, networkPolicy.Spec.Egress, 2)
	})

	t.Run("specifying ingresses allows traffic from istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
			"https://gief.api.plz",
		}
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIpsOptions)
		assert.NotNil(t, networkPolicy)
		assert.Len(t, networkPolicy.Spec.Ingress[0].From, 1)

		podMatch := map[string]string{"istio": "ingressgateway"}
		namespaceMatch := map[string]string{"name": "istio-system"}

		assert.Equal(t, podMatch, networkPolicy.Spec.Ingress[1].From[0].PodSelector.MatchLabels)
		assert.Equal(t, namespaceMatch, networkPolicy.Spec.Ingress[1].From[0].NamespaceSelector.MatchLabels)
	})

	t.Run("specifying ingresses when all traffic is allowed still creates an explicit rule for istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
			"https://gief.api.plz",
		}
		app.Spec.AccessPolicy.Inbound.Rules = append(app.Spec.AccessPolicy.Inbound.Rules, nais_io_v1.AccessPolicyRule{Application: "*"})
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIpsOptions)
		assert.NotNil(t, networkPolicy)
		assert.Len(t, networkPolicy.Spec.Ingress, 3)
		assert.Len(t, networkPolicy.Spec.Ingress[0].From, 1)
		assert.Len(t, networkPolicy.Spec.Ingress[1].From, 1)

		istioPodMatch := map[string]string{"istio": "ingressgateway"}
		istioNamespaceMatch := map[string]string{"name": "istio-system"}
		prometheusMatch := map[string]string{"app": "prometheus"}
		asterix := networking.NetworkPolicyIngressRule{
			Ports: nil,
			From: []networking.NetworkPolicyPeer{{
				PodSelector:       &metav1.LabelSelector{},
				NamespaceSelector: nil,
				IPBlock:           nil,
			}},
		}
		assert.Equal(t, prometheusMatch, networkPolicy.Spec.Ingress[0].From[0].PodSelector.MatchLabels)
		assert.Equal(t, istioNamespaceMatch, networkPolicy.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels)
		assert.Equal(t, asterix, networkPolicy.Spec.Ingress[1])
		assert.Equal(t, istioPodMatch, networkPolicy.Spec.Ingress[2].From[0].PodSelector.MatchLabels)
		assert.Equal(t, istioNamespaceMatch, networkPolicy.Spec.Ingress[2].From[0].NamespaceSelector.MatchLabels)
	})
	t.Run("all traffic inside namespace sets from rule to empty podspec", func(t *testing.T) {
		app := fixtures.MinimalApplication()

		app.Spec.AccessPolicy.Inbound.Rules = append(app.Spec.AccessPolicy.Inbound.Rules, nais_io_v1.AccessPolicyRule{Application: "*"})
		err := nais_io_v1alpha1.ApplyDefaults(app)
		assert.NoError(t, err)

		networkPolicy := resourcecreator.NetworkPolicy(app, defaultIpsOptions)
		assert.NotNil(t, networkPolicy)

		yamlres, err := yaml.Marshal(networkPolicy)
		assert.NotNil(t, yamlres)
		assert.Empty(t, networkPolicy.Spec.Ingress[1].From[0].PodSelector)
	})

}
