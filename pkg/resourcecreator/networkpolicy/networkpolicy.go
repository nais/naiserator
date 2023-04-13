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
	GetAivenRange() string
	GetGatewayMappings() []config.GatewayMapping
	GetGoogleProjectID() string
	GetNaisNamespace() string
	IsNetworkPolicyEnabled() bool
	IsLegacyGCP() bool
}

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
		np.Spec = legacyNetpolSpec(source.GetName(), cfg.GetClusterName())
		np.SetName(source.GetName() + "-legacy")
		ast.AppendOperation(resource.OperationCreateOrUpdate, np)
	}

	np := baseNetworkPolicy(source)
	np.Spec = netpolSpec(source.GetName(), cfg, source.GetAccessPolicy(), source.GetIngress(), source.GetLeaderElection())
	ast.AppendOperation(resource.OperationCreateOrUpdate, np)
}

func netpolSpec(name string, cfg Config, policy *nais_io_v1.AccessPolicy, ingress []nais_io_v1.Ingress, election bool) networkingv1.NetworkPolicySpec {
	return networkingv1.NetworkPolicySpec{
		PodSelector: *labelSelector("app", name),
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: ingressRules(ingress, policy, cfg),
		Egress:  egressRules(policy, cfg, election),
	}
}

func egressRules(policy *nais_io_v1.AccessPolicy, cfg Config, election bool) []networkingv1.NetworkPolicyEgressRule {
	rules := make([]networkingv1.NetworkPolicyEgressRule, 0)

	rules = append(rules, defaultEgressRules(cfg)...)
	rules = append(rules, egressRulesFromAccessPolicy(policy, cfg)...)
	if election {
		rules = append(rules, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: cfg.GetAPIServerIP(),
					},
				},
			},
		})
	}

	return rules
}

func egressRulesFromAccessPolicy(policy *nais_io_v1.AccessPolicy, cfg Config) []networkingv1.NetworkPolicyEgressRule {
	if policy == nil || policy.Outbound == nil || len(policy.Outbound.Rules) == 0 {
		return nil
	}

	peers := make([]networkingv1.NetworkPolicyPeer, 0)
	for _, rule := range policy.Outbound.Rules.GetRules() {
		// non-local access policy rules do not result in network policies
		if rule.Application == "" || rule.Application == "*" || !rule.MatchesCluster(cfg.GetClusterName()) {
			continue
		}

		peer := networkingv1.NetworkPolicyPeer{
			PodSelector: labelSelector("app", rule.Application),
		}

		if rule.Namespace != "" {
			peer.NamespaceSelector = labelSelector("name", rule.Namespace)
		}

		peers = append(peers, peer)
	}

	if len(peers) == 0 {
		return nil
	}

	return []networkingv1.NetworkPolicyEgressRule{
		{
			To: peers,
		},
	}
}

func defaultEgressRules(cfg Config) []networkingv1.NetworkPolicyEgressRule {
	rules := []networkingv1.NetworkPolicyEgressRule{
		{
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{},
					PodSelector:       labelSelector("k8s-app", "kube-dns"),
				},
			},
		},
	}
	if cfg.GetAivenRange() != "" {
		rules = append(rules, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: cfg.GetAivenRange(),
					},
				},
			},
		})
	}
	return rules
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
				// assumes that ingressClass equals instance name label
				rules = append(rules, networkingv1.NetworkPolicyIngressRule{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: labelSelector("name", "nginx"),
							PodSelector:       labelSelector("app.kubernetes.io/instance", *ingressClass),
						},
					},
				})
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
					PodSelector:       labelSelector("app.kubernetes.io/name", "prometheus"),
				},
			},
		},
	}
}

func legacyNetpolSpec(appName string, clusterName string) networkingv1.NetworkPolicySpec {
	// TODO: remove when all clusters are migrated to new network policies
	if clusterName != "prod-gcp" {
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
			},
		}
	} else {
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

		if rule.Namespace == "*" {
			peer.NamespaceSelector = &metav1.LabelSelector{}
		} else if rule.Namespace != "" {
			peer.NamespaceSelector = labelSelector("name", rule.Namespace)
		}

		peers = append(peers, peer)
	}

	if len(peers) == 0 {
		return nil
	}

	return []networkingv1.NetworkPolicyIngressRule{
		{
			From: peers,
		},
	}
}

func labelSelector(label string, value string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			label: value,
		},
	}
}
