package ciliumnetworkpolicy

import (
	"net/url"

	cilium_io_v2 "github.com/nais/liberator/pkg/apis/cilium.io/v2"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	prometheusPodSelectorLabelValue        = "prometheus" // Label value denoting the Prometheus pod-selector
	prometheusNamespace                    = "nais"       // Which namespace Prometheus is installed in
	nginxNamespace                         = "nginx"      // Which namespace Nginx ingress controller runs in
	networkPolicyDefaultEgressAllowIPBlock = "0.0.0.0/0"  // The default IP block CIDR for the default allow network policies per app

	namespaceSelector = "io.kubernetes.pod.namespace"
)

func Create(source Source, ast *resource.Ast, cfg Config) {
	if !cfg.IsNetworkPolicyEnabled() {
		return
	}

	networkPolicy := &cilium_io_v2.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
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

func networkPolicySpec(appName string, options Config, naisAccessPolicy *nais_io_v1.AccessPolicy, naisIngresses []nais_io_v1.Ingress, leaderElection bool) cilium_io_v2.NetworkPolicySpec {
	return cilium_io_v2.NetworkPolicySpec{
		EndpointSelector: labelSelector("app", appName),
		Ingress:          ingressPolicy(options, naisAccessPolicy.Inbound, naisIngresses),
		Egress:           egressPolicy(options, naisAccessPolicy.Outbound, leaderElection),
	}
}

func networkPolicyApplicationRules(rules nais_io_v1.AccessPolicyBaseRules, options Config) (networkPolicy []cilium_io_v2.Ingress) {
	for _, rule := range rules.GetRules() {

		// non-local access policy rules do not result in network policies
		if !rule.MatchesCluster(options.GetClusterName()) {
			continue
		}

		networkPolicyPeer := cilium_io_v2.Ingress{
			FromEndpoints: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{
						"app": rule.Application,
					},
				},
			},
		}

		if rule.Application == "*" {
			networkPolicyPeer = cilium_io_v2.Ingress{FromEndpoints: []*metav1.LabelSelector{}}
		}

		if rule.Namespace != "" {
			if len(networkPolicyPeer.FromEndpoints) > 0 {
				if networkPolicyPeer.FromEndpoints[0].MatchLabels == nil {
					networkPolicyPeer.FromEndpoints[0].MatchLabels = map[string]string{}
				}
			} else {
				networkPolicyPeer.FromEndpoints = []*metav1.LabelSelector{
					{
						MatchLabels: map[string]string{},
					},
				}
			}
			networkPolicyPeer.FromEndpoints[0].MatchLabels[namespaceSelector] = rule.Namespace
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}

func ingressPolicy(options Config, naisAccessPolicyInbound *nais_io_v1.AccessPolicyInbound, naisIngresses []nais_io_v1.Ingress) []cilium_io_v2.Ingress {
	rules := make([]cilium_io_v2.Ingress, 0)

	rules = append(rules, ingressPeer("app", prometheusPodSelectorLabelValue, prometheusNamespace))

	if len(naisAccessPolicyInbound.Rules) > 0 {
		rules = append(rules, networkPolicyApplicationRules(naisAccessPolicyInbound.Rules, options)...)
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
			nginxNamespace := nginxNamespace
			instance := *gw
			if options.IsNaisSystemEnabled() {
				nginxNamespace = "nais-system"
				instance = "loadbalancer"
			}
			rules = append(rules, ingressPeer("app.kubernetes.io/instance", instance, nginxNamespace))
		}
	}

	return rules
}

func ingressPeer(labelName, labelValue, namespace string) cilium_io_v2.Ingress {
	return cilium_io_v2.Ingress{
		FromEndpoints: []*metav1.LabelSelector{
			{
				MatchLabels: map[string]string{
					labelName:         labelValue,
					namespaceSelector: namespace,
				},
			},
		},
	}
}

func egressPolicy(options Config, naisAccessPolicyOutbound *nais_io_v1.AccessPolicyOutbound, leaderElection bool) []cilium_io_v2.Egress {
	defaultRules := defaultAllowEgress(options)

	if len(naisAccessPolicyOutbound.Rules) > 0 {
		rules := networkPolicyApplicationRules(naisAccessPolicyOutbound.Rules, options)
		for _, rule := range rules {
			defaultRules = append(defaultRules, cilium_io_v2.Egress{ToEndpoints: rule.FromEndpoints})
		}
	}

	for _, ext := range naisAccessPolicyOutbound.External {
		u, err := url.Parse(ext.Host)
		if err != nil {
			continue
		}

		host := u.Host
		if host == "" {
			host = u.Path
		}

		defaultRules = append(defaultRules, cilium_io_v2.Egress{
			ToFQDNs: []cilium_io_v2.FQDN{
				{
					MatchName: host,
				},
			},
		})
	}

	if leaderElection && len(options.GetGoogleProjectID()) > 0 {
		defaultRules = append(defaultRules, cilium_io_v2.Egress{
			ToCIDRSet: []cilium_io_v2.CIDRSet{
				{
					CIDR: options.GetAPIServerIP(),
				},
			},
		})
	}

	return defaultRules
}

func defaultAllowEgress(options Config) []cilium_io_v2.Egress {
	return []cilium_io_v2.Egress{
		{
			ToEndpoints: []*metav1.LabelSelector{
				{
					MatchLabels: map[string]string{
						"io.kubernetes.pod.namespace": "kube-system",
						"k8s-app":                     "kube-dns",
					},
				},
			},
		},
		{
			ToCIDRSet: []cilium_io_v2.CIDRSet{
				{
					CIDR:   networkPolicyDefaultEgressAllowIPBlock,
					Except: options.GetAccessPolicyNotAllowedCIDRs(),
				},
			},
		},
	}
}
