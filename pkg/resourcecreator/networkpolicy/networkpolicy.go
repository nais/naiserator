package networkpolicy

import (
	"net/url"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
)

type Source interface {
	resource.Source
	GetAccessPolicy() *nais_io_v1.AccessPolicy
	GetIngress() []nais_io_v1.Ingress
	GetLeaderElection() bool
}

type Config interface {
	GetAPIServerIP() string
	GetAccessPolicyNotAllowedCIDRs() []string
	GetClusterName() string
	GetGatewayMappings() []config.GatewayMapping
	GetGoogleProjectID() string
	GetNaisNamespace() string
	IsNaisSystemEnabled() bool
	IsNetworkPolicyEnabled() bool
	IsLegacyGCP() bool
}

const (
	prometheusPodSelectorLabelValue = "prometheus" // Label value denoting the Prometheus pod-selector
)

func baseNetworkPolicy(source Source) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
	}
}

func Create(source Source, ast *resource.Ast, cfg Config) {
	if !cfg.IsNetworkPolicyEnabled() {
		return
	}

	if cfg.IsLegacyGCP() {
		np := baseNetworkPolicy(source)
		np.Spec = legacyNetpolSpec(source.GetName())
		np.SetName(source.GetName() + "-legacy")
		ast.AppendOperation(resource.OperationCreateOrUpdate, np)
	}

	np := baseNetworkPolicy(source)
	np.Spec = netpolSpec(source.GetName(), cfg, source.GetAccessPolicy(), source.GetIngress(), source.GetLeaderElection())

	// # outbound network policy
	// - kube-dns
	// - if (leaderelection) apiserver
	// - if (accesspolicy) accesspolicies
	// - aiven range private, fra config (via chart, via fasit mapping values fra nais-terraform-modules) cfg.Features.AivenRange
	//
	// # inbound network policy
	// - prometheus
	// - if (accesspolicy) accesspolicies
	// - if (ingress) ingresscontroller (sjekk diff her på legacy vs naas)
	// - policy for at kubelet skal kunne kalle helsesjekk

	//  egress:
	//   - namespaceSelector: {}
	// 	   podSelector:
	// 	     matchLabels:
	// 		   k8s-app: kube-dns
	// ingress:
	//   - from:
	// - namespaceSelector:
	//     matchLabels:
	//       name: nais-system
	//   podSelector:
	//     matchLabels:
	//       app.kubernetes.io/name: prometheus

}

func netpolSpec(name string, cfg Config, policy *nais_io_v1.AccessPolicy, ingress []nais_io_v1.Ingress, election bool) networkingv1.NetworkPolicySpec {
	return networkingv1.NetworkPolicySpec{
		PodSelector: *labelSelector("app", name),
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: ingressRules(ingress, policy, cfg),
	}
}

func ingressRules(ingress []nais_io_v1.Ingress, policy *nais_io_v1.AccessPolicy, cfg Config) []networkingv1.NetworkPolicyIngressRule {
	rules := make([]networkingv1.NetworkPolicyIngressRule, 0)

	rules = append(rules, defaultIngressRules(cfg)...)
	rules = append(rules, ingressRulesFromIngress(ingress, cfg)...)
	rules = append(rules, ingressRulesFromAccessPolicy(policy, cfg)...)

	return rules
}

func ingressRulesFromIngress(ingress []nais_io_v1.Ingress, cfg Config) []networkingv1.NetworkPolicyIngressRule {
	rules := make([]networkingv1.NetworkPolicyIngressRule, 0)
	if len(ingress) > 0 {
		for _, ingress := range ingress {
			ur, err := url.Parse(string(ingress))
			if err != nil {
				continue
			}
			ingressClass := util.ResolveIngressClass(ur.Host, cfg.GetGatewayMappings())
			if ingressClass == nil {
				continue
			}

			ls := labelSelector("nais.io/ingressClass", *ingressClass)
			ns := cfg.GetNaisNamespace()

			// TODO: remove when loadbalancer features are installed in nais-system for legacy gcp
			if cfg.IsLegacyGCP() {
				ns = "nginx"
				// assumes that ingressClass equals instance name label
				ls = labelSelector("app.kubernetes.io/instance", *ingressClass)
			}

			rules = append(rules, networkingv1.NetworkPolicyIngressRule{
				From: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: labelSelector("name", ns),
						PodSelector:       ls,
					},
				},
			})
		}
	}
	return rules
}

func defaultIngressRules(cfg Config) []networkingv1.NetworkPolicyIngressRule {
	return []networkingv1.NetworkPolicyIngressRule{
		{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: labelSelector("name", cfg.GetNaisNamespace()),
					PodSelector:       labelSelector("app.kubernetes.io/name", prometheusPodSelectorLabelValue),
				},
			},
		},
	}
}

func legacyNetpolSpec(appName string) networkingv1.NetworkPolicySpec {
	return networkingv1.NetworkPolicySpec{
		PodSelector: *labelSelector("app", appName),
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: []networkingv1.NetworkPolicyIngressRule{
			{
				From: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: labelSelector("linkerd.io/is-control-plane", "true"),
					},
				},
			},
		},
		Egress: []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: labelSelector("linkerd.io/is-control-plane", "true"),
				}},
			},
			{
				To: []networkingv1.NetworkPolicyPeer{{
					IPBlock: &networkingv1.IPBlock{
						CIDR:   "0.0.0.0/0",
						Except: []string{"10.6.0.0/15", "172.16.0.0/12", "192.168.0.0/16"},
					}},
				},
			},
		},
	}
}

func ingressRulesFromAccessPolicy(policy *nais_io_v1.AccessPolicy, options Config) []networkingv1.NetworkPolicyIngressRule {
	if policy == nil || policy.Inbound == nil || len(policy.Inbound.Rules) == 0 {
		return nil
	}

	peers := make([]networkingv1.NetworkPolicyPeer, 0)
	for _, rule := range policy.Inbound.Rules.GetRules() {
		// non-local access policy rules do not result in network policies
		if !rule.MatchesCluster(options.GetClusterName()) {
			continue
		}

		peer := networkingv1.NetworkPolicyPeer{}
		if rule.Application == "*" {
			peer.PodSelector = &metav1.LabelSelector{}
		} else {
			peer.PodSelector = labelSelector("app", rule.Application)
		}

		if rule.Namespace != "" {
			peer.NamespaceSelector = labelSelector("name", rule.Namespace)
		}

		peers = append(peers, peer)
	}

	return []networkingv1.NetworkPolicyIngressRule{
		{
			From: peers,
		},
	}
}

func egressPolicy(options Config, naisAccessPolicyOutbound *nais_io_v1.AccessPolicyOutbound, leaderElection bool) []networkingv1.NetworkPolicyEgressRule {
	appRules := networkPolicyApplicationRules(naisAccessPolicyOutbound.Rules, options)

	if len(appRules) > 0 {
		appEgressRule := networkPolicyEgressRule(appRules...)
		defaultRules = append(defaultRules, appEgressRule)
	}

	if leaderElection && len(options.GetGoogleProjectID()) > 0 {
		apiServerAccessRule := networkPolicyEgressRule(networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: options.GetAPIServerIP(),
			},
		})
		defaultRules = append(defaultRules, apiServerAccessRule)
	}

	return defaultRules
}

func labelSelector(label string, value string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			label: value,
		},
	}
}
