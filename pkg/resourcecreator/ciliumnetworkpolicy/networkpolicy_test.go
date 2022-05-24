package ciliumnetworkpolicy_test

import (
	"testing"

	cilium_io_v2 "github.com/nais/liberator/pkg/apis/cilium.io/v2"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/resourcecreator/ciliumnetworkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const accessPolicyApp = "allowedAccessApp"

var defaultIps = []string{"12.0.0.0/12", "123.0.0.0/12"}

func TestNetworkPolicy(t *testing.T) {
	opts := &generators.Options{}
	opts.Config.Features.NetworkPolicy = true
	opts.Config.Features.AccessPolicyNotAllowedCIDRs = defaultIps

	t.Run("cilium default deny all sets app rules to empty slice", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		ciliumnetworkpolicy.Create(app, ast, opts)
		networkPolicy := ast.Operations[0].Resource.(*cilium_io_v2.NetworkPolicy)

		assert.Len(t, networkPolicy.Spec.Egress[0].ToEndpoints, 1)

		testPolicy := make([]cilium_io_v2.Ingress, 0)

		testPolicy = append(testPolicy, cilium_io_v2.Ingress{
			FromEndpoints: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{
						"app":                         "prometheus",
						"io.kubernetes.pod.namespace": "nais",
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

		ciliumnetworkpolicy.Create(app, ast, opts)
		networkPolicy := ast.Operations[0].Resource.(*cilium_io_v2.NetworkPolicy)

		matchLabels := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": accessPolicyApp,
			},
		}

		assert.Equal(t, matchLabels, networkPolicy.Spec.Egress[2].ToEndpoints[0])
	})

	t.Run("allowed app in egress rule sets egress app rules and default rules", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.AccessPolicy.Outbound.Rules = append(app.Spec.AccessPolicy.Outbound.Rules, nais_io_v1.AccessPolicyRule{Application: accessPolicyApp})
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		ciliumnetworkpolicy.Create(app, ast, opts)
		networkPolicy := ast.Operations[0].Resource.(*cilium_io_v2.NetworkPolicy)
		matchLabels0 := []*metav1.LabelSelector{
			{
				MatchLabels: map[string]string{
					"io.kubernetes.pod.namespace": "kube-system",
					"k8s-app":                     "kube-dns",
				},
			},
		}

		matchLabels2 := []*metav1.LabelSelector{
			{
				MatchLabels: map[string]string{
					"app": accessPolicyApp,
				},
			},
		}

		assert.Equal(t, matchLabels0, networkPolicy.Spec.Egress[0].ToEndpoints)
		assert.Equal(t, matchLabels2, networkPolicy.Spec.Egress[2].ToEndpoints)
	})

	t.Run("all traffic inside namespace sets from rule to empty podspec", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		rule := nais_io_v1.AccessPolicyInboundRule{AccessPolicyRule: nais_io_v1.AccessPolicyRule{Application: "*"}}
		app.Spec.AccessPolicy.Inbound.Rules = append(app.Spec.AccessPolicy.Inbound.Rules, rule)
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		ciliumnetworkpolicy.Create(app, ast, opts)
		networkPolicy := ast.Operations[0].Resource.(*cilium_io_v2.NetworkPolicy)
		assert.NotNil(t, networkPolicy)

		yamlres, err := yaml.Marshal(networkPolicy)
		assert.NoError(t, err)
		assert.NotNil(t, yamlres)
		assert.Empty(t, networkPolicy.Spec.Ingress[1].FromEndpoints)
	})
}
