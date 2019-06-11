package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addNetworkPolicyGressRules(rules []nais.AccessPolicyGressRule) (networkPolicy []networkingv1.NetworkPolicyPeer) {
	for _, gress := range rules {
		networkPolicyPeer := networkingv1.NetworkPolicyPeer{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": gress.Application,
				},
			},
		}

		if gress.Namespace != "" {
			networkPolicyPeer.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": gress.Namespace,
				},
			}
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}

func ingressPolicy(app *nais.Application) []networkingv1.NetworkPolicyIngressRule {
	if app.Spec.AccessPolicy.Ingress.AllowAll {
		return []networkingv1.NetworkPolicyIngressRule{{}}
	}

	if len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		return []networkingv1.NetworkPolicyIngressRule{
			{
				From: addNetworkPolicyGressRules(app.Spec.AccessPolicy.Ingress.Rules),
			},
		}
	}

	return []networkingv1.NetworkPolicyIngressRule{}
}

func egressPolicy(app *nais.Application) []networkingv1.NetworkPolicyEgressRule {
	if app.Spec.AccessPolicy.Egress.AllowAll {
		return []networkingv1.NetworkPolicyEgressRule{{}}
	}

	if len(app.Spec.AccessPolicy.Egress.Rules) > 0 {
		return []networkingv1.NetworkPolicyEgressRule{
			{
				To: addNetworkPolicyGressRules(app.Spec.AccessPolicy.Egress.Rules),
			},
		}
	}

	return []networkingv1.NetworkPolicyEgressRule{}
}

func networkPolicySpec(app *nais.Application) *networkingv1.NetworkPolicySpec {
	return &networkingv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": app.Name,
			},
		},
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: ingressPolicy(app),
		Egress:  egressPolicy(app),
	}

}

func NetworkPolicy(app *nais.Application) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec:       *networkPolicySpec(app),
	}
}
