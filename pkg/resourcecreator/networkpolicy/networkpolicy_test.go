package networkpolicy_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const accessPolicyApp = "allowedAccessApp"

var defaultIps = []string{"12.0.0.0/12", "123.0.0.0/12"}

func TestNetworkPolicy(t *testing.T) {

	resourceOptions := resource.NewOptions()
	resourceOptions.NetworkPolicy = true
	resourceOptions.AccessPolicyNotAllowedCIDRs = defaultIps

	t.Run("default deny all sets app rules to empty slice", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
		networkPolicy := ast.Operations[0].Resource.(*networking.NetworkPolicy)

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
							"name": "nais",
						},
					},
				},
			},
		})

		testPolicy = append(testPolicy, networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"linkerd.io/is-control-plane": "true",
						},
					},
				},
			},
		})

		testPolicy = append(testPolicy, networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"component": "tap",
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"linkerd.io/extension": "viz",
						},
					},
				},
			},
		})

		testPolicy = append(testPolicy, networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"component": "prometheus",
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"linkerd.io/extension": "viz",
						},
					},
				},
			},
		})

		assert.Equal(t, testPolicy, networkPolicy.Spec.Ingress)
	})

	t.Run("allowed app in egress rule sets network policy pod selector to allowed app", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais_io_v1.AccessPolicyRule{Application: accessPolicyApp})
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
		networkPolicy := ast.Operations[0].Resource.(*networking.NetworkPolicy)

		matchLabels := map[string]string{
			"app": accessPolicyApp,
		}

		assert.Equal(t, matchLabels, networkPolicy.Spec.Egress[1].To[0].PodSelector.MatchLabels)
	})

	t.Run("allowed app in egress rule sets egress app rules and default rules", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais_io_v1.AccessPolicyRule{Application: accessPolicyApp})
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
		networkPolicy := ast.Operations[0].Resource.(*networking.NetworkPolicy)

		assert.Len(t, networkPolicy.Spec.Egress, 2)
	})

	t.Run("all traffic inside namespace sets from rule to empty podspec", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.AccessPolicy.Inbound.Rules = append(app.Spec.AccessPolicy.Inbound.Rules, nais_io_v1.AccessPolicyRule{Application: "*"})
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
		networkPolicy := ast.Operations[0].Resource.(*networking.NetworkPolicy)
		assert.NotNil(t, networkPolicy)

		yamlres, err := yaml.Marshal(networkPolicy)
		assert.NotNil(t, yamlres)
		assert.Empty(t, networkPolicy.Spec.Ingress[1].From[0].PodSelector)
	})

}
