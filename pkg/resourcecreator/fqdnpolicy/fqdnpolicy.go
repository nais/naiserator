package fqdnpolicy

import (
	"strings"

	fqdn "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha2"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const defaultPort = 443

type Config interface {
	GetFQDNPolicy() config.FQDNPolicy
	IsNetworkPolicyEnabled() bool
}

func Create(source networkpolicy.Source, ast *resource.Ast, cfg Config) {
	if !(cfg.IsNetworkPolicyEnabled() && cfg.GetFQDNPolicy().Enabled) {
		return
	}

	meta := resource.CreateObjectMeta(source)
	meta.SetName(source.GetName() + "-fqdn")
	policy := &fqdn.FQDNNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FQDNNetworkPolicy",
			APIVersion: "networking.gke.io/v1alpha2",
		},
		ObjectMeta: meta,
		Spec:       fqdnPolicySpec(cfg, source.GetName(), source.GetAccessPolicy()),
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, policy)
}

func fqdnPolicySpec(cfg Config, name string, policy *nais_io_v1.AccessPolicy) fqdn.FQDNNetworkPolicySpec {
	merged := append(defaultEgressPolicy(cfg), egressPolicy(policy.Outbound)...)
	return fqdn.FQDNNetworkPolicySpec{
		PodSelector: *labelSelector("app", name),
		Egress:      merged,
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeEgress,
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

func defaultEgressPolicy(cfg Config) []fqdn.FQDNNetworkPolicyEgressRule {
	if len(cfg.GetFQDNPolicy().Rules) == 0 {
		return nil
	}

	var rules []fqdn.FQDNNetworkPolicyEgressRule
	for _, r := range cfg.GetFQDNPolicy().Rules {
		if r.Port == 0 || r.Host == "" {
			log.Errorf("Invalid FQDN policy rule in config: %v", r)
			continue
		}
		port := &[]intstr.IntOrString{intstr.FromInt(r.Port)}[0]
		rules = append(rules, fqdn.FQDNNetworkPolicyEgressRule{
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &[]v1.Protocol{v1.ProtocolTCP}[0],
					Port:     port,
				},
			},
			To: []fqdn.FQDNNetworkPolicyPeer{
				{
					FQDNs: []string{r.Host},
				},
			},
		})
	}
	return rules
}

func egressPolicy(outbound *nais_io_v1.AccessPolicyOutbound) []fqdn.FQDNNetworkPolicyEgressRule {
	var rules []fqdn.FQDNNetworkPolicyEgressRule
	for _, e := range outbound.External {
		var np []networkingv1.NetworkPolicyPort
		if e.Ports == nil {
			np = []networkingv1.NetworkPolicyPort{defaultNetworkPolicyPort()}
		} else {
			np = networkPolicyPorts(e.Ports)
		}

		host := strings.Replace(strings.Replace(e.Host, "https://", "", 1), "http://", "", 1)
		rules = append(rules, fqdn.FQDNNetworkPolicyEgressRule{
			Ports: np,
			To: []fqdn.FQDNNetworkPolicyPeer{
				{
					FQDNs: []string{host},
				},
			},
		})
	}
	return rules
}

func networkPolicyPorts(rules []nais_io_v1.AccessPolicyPortRule) []networkingv1.NetworkPolicyPort {
	var ports []networkingv1.NetworkPolicyPort
	for _, rule := range rules {
		ports = append(ports, networkPolicyPort(&rule))
	}
	return ports
}

func networkPolicyPort(rule *nais_io_v1.AccessPolicyPortRule) networkingv1.NetworkPolicyPort {
	np := defaultNetworkPolicyPort()
	if rule.Port != 0 {
		port := intstr.FromInt(int(rule.Port))
		np.Port = &port
	}
	return np
}

func defaultNetworkPolicyPort() networkingv1.NetworkPolicyPort {
	proto := v1.ProtocolTCP
	port := intstr.FromInt(defaultPort)
	return networkingv1.NetworkPolicyPort{
		Protocol: &proto,
		Port:     &port,
	}
}
