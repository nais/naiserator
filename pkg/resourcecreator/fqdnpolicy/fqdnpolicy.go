package fqdnpolicy

import (
	fqdn "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha2"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Config interface {
	IsFQDNPolicyEnabled() bool
	IsNetworkPolicyEnabled() bool
}

func Create(source networkpolicy.Source, ast *resource.Ast, cfg Config) {
	if !cfg.IsNetworkPolicyEnabled() || !cfg.IsFQDNPolicyEnabled() {
		return
	}

	policy := &fqdn.FQDNNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FQDNNetworkPolicy",
			APIVersion: "networking.gke.io/v1alpha2",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec:       fqdnPolicySpec(source.GetName(), source.GetAccessPolicy()),
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, policy)
}

func fqdnPolicySpec(name string, policy *nais_io_v1.AccessPolicy) fqdn.FQDNNetworkPolicySpec {
	return fqdn.FQDNNetworkPolicySpec{
		PodSelector: *labelSelector("app", name),
		Egress:      egressPolicy(policy.Outbound),
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

func egressPolicy(outbound *nais_io_v1.AccessPolicyOutbound) []fqdn.FQDNNetworkPolicyEgressRule {
	var rules []fqdn.FQDNNetworkPolicyEgressRule
	for _, e := range outbound.External {
		rules = append(rules, fqdn.FQDNNetworkPolicyEgressRule{
			Ports: networkPolicyPorts(e.Ports),
			To: []fqdn.FQDNNetworkPolicyPeer{
				{
					FQDNs: []string{e.Host},
				},
			},
		})
	}
	return rules
}

func networkPolicyPorts(rules []nais_io_v1.AccessPolicyPortRule) []networkingv1.NetworkPolicyPort {
	var ports []networkingv1.NetworkPolicyPort
	for _, rule := range rules {
		proto := v1.ProtocolTCP
		port := intstr.FromInt(int(rule.Port))
		ports = append(ports, networkingv1.NetworkPolicyPort{
			Protocol: &proto,
			Port:     &port,
		})
	}
	return ports
}
