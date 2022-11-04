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
	IsNaisSystemEnabled() bool
	IsNetworkPolicyEnabled() bool
}

const (
	prometheusPodSelectorLabelValue        = "prometheus"  // Label value denoting the Prometheus pod-selector
	prometheusNamespace                    = "nais"        // Which namespace Prometheus is installed in
	prometheusNaasNamespace                = "nais-system" // Which namespace Prometheus is installed in naas-clusters
	ingressControllerNamespace             = "nginx"       // Which namespace Nginx ingress controller runs in
	networkPolicyDefaultEgressAllowIPBlock = "0.0.0.0/0"   // The default IP block CIDR for the default allow network policies per app
)

func Create(source Source, ast *resource.Ast, cfg Config) {
	if !cfg.IsNetworkPolicyEnabled() {
		return
	}

	networkPolicy := &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec:       networkPolicySpec(source.GetName(), cfg, source.GetAccessPolicy(), source.GetIngress(), source.GetLeaderElection()),
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, networkPolicy)
}

func labelSelector(label string, value string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			label: value,
		},
	}
}

func networkPolicySpec(appName string, options Config, naisAccessPolicy *nais_io_v1.AccessPolicy, naisIngresses []nais_io_v1.Ingress, leaderElection bool) networkingv1.NetworkPolicySpec {
	return networkingv1.NetworkPolicySpec{
		PodSelector: *labelSelector("app", appName),
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: ingressPolicy(options, naisAccessPolicy.Inbound, naisIngresses),
		Egress:  egressPolicy(options, naisAccessPolicy.Outbound, leaderElection),
	}
}

func networkPolicyPeer(podLabelName, podLabelValue, namespace string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: labelSelector("name", namespace),
		PodSelector:       labelSelector(podLabelName, podLabelValue),
	}
}

func networkPolicyIngressRule(peer ...networkingv1.NetworkPolicyPeer) networkingv1.NetworkPolicyIngressRule {
	return networkingv1.NetworkPolicyIngressRule{
		From: peer,
	}
}

func networkPolicyEgressRule(peer ...networkingv1.NetworkPolicyPeer) networkingv1.NetworkPolicyEgressRule {
	return networkingv1.NetworkPolicyEgressRule{
		To: peer,
	}
}

func networkPolicyApplicationRules(rules nais_io_v1.AccessPolicyBaseRules, options Config) (networkPolicy []networkingv1.NetworkPolicyPeer) {
	if len(rules.GetRules()) == 0 {
		return
	}

	for _, rule := range rules.GetRules() {

		// non-local access policy rules do not result in network policies
		if !rule.MatchesCluster(options.GetClusterName()) {
			continue
		}

		networkPolicyPeer := networkingv1.NetworkPolicyPeer{
			PodSelector: labelSelector("app", rule.Application),
		}

		if rule.Application == "*" {
			networkPolicyPeer = networkingv1.NetworkPolicyPeer{PodSelector: &metav1.LabelSelector{}}
		}

		if rule.Namespace != "" {
			networkPolicyPeer.NamespaceSelector = labelSelector("name", rule.Namespace)
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}

func ingressPolicy(options Config, naisAccessPolicyInbound *nais_io_v1.AccessPolicyInbound, naisIngresses []nais_io_v1.Ingress) []networkingv1.NetworkPolicyIngressRule {
	rules := make([]networkingv1.NetworkPolicyIngressRule, 0)

	rules = append(rules, networkPolicyIngressRule(networkPolicyPeer("app", prometheusPodSelectorLabelValue, prometheusNamespace)))
	rules = append(rules, networkPolicyIngressRule(networkingv1.NetworkPolicyPeer{
		NamespaceSelector: labelSelector("linkerd.io/is-control-plane", "true"),
	}))
	rules = append(rules, networkPolicyIngressRule(networkPolicyPeer("app.kubernetes.io/name", prometheusPodSelectorLabelValue, prometheusNaasNamespace)))

	appRules := networkPolicyApplicationRules(naisAccessPolicyInbound.Rules, options)

	if len(appRules) > 0 {
		appIngressRule := networkPolicyIngressRule(appRules...)
		rules = append(rules, appIngressRule)
	}

	if len(naisIngresses) > 0 {
		for _, ingress := range naisIngresses {
			ur, err := url.Parse(string(ingress))
			if err != nil {
				continue
			}
			gw := util.ResolveIngressClass(ur.Host, options.GetGatewayMappings())
			if gw == nil {
				continue
			}
			ingressControllerNamespace := ingressControllerNamespace
			instance := *gw
			if options.IsNaisSystemEnabled() {
				ingressControllerNamespace = "nais-system"
			}
			// assumes that ingressClass equals instance name label
			rules = append(rules, networkPolicyIngressRule(networkingv1.NetworkPolicyPeer{
				PodSelector:       labelSelector("app.kubernetes.io/instance", instance),
				NamespaceSelector: labelSelector("name", ingressControllerNamespace),
			}))
		}
	}

	return rules
}

func egressPolicy(options Config, naisAccessPolicyOutbound *nais_io_v1.AccessPolicyOutbound, leaderElection bool) []networkingv1.NetworkPolicyEgressRule {
	defaultRules := defaultAllowEgress(options)
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

func defaultAllowEgress(options Config) []networkingv1.NetworkPolicyEgressRule {
	peers := make([]networkingv1.NetworkPolicyPeer, 0, 4)

	peers = append(peers, networkingv1.NetworkPolicyPeer{
		NamespaceSelector: labelSelector("linkerd.io/is-control-plane", "true"),
	})

	peers = append(peers, networkingv1.NetworkPolicyPeer{
		PodSelector: labelSelector("k8s-app", "kube-dns"),
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				// select in all namespaces since labels on kube-system is regularly deleted in GCP
			},
		},
	})

	peers = append(peers, networkingv1.NetworkPolicyPeer{
		IPBlock: &networkingv1.IPBlock{
			CIDR:   networkPolicyDefaultEgressAllowIPBlock,
			Except: options.GetAccessPolicyNotAllowedCIDRs(),
		},
	})

	return []networkingv1.NetworkPolicyEgressRule{
		networkPolicyEgressRule(peers...),
	}
}
