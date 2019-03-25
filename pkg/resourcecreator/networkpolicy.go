package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addNetworkPolicyIngressRules(app *nais.Application) (networkPolicy []networkingv1.NetworkPolicyPeer) {
	for _, ingress := range app.Spec.AccessPolicy.Ingress.Rules {
		networkPolicyPeer := networkingv1.NetworkPolicyPeer{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": ingress.Application,
				},
			},
		}

		if ingress.Namespace != "" {
			networkPolicyPeer.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": ingress.Namespace,
				},
			}
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}


func ingressPolicy(app *nais.Application) *[]networkingv1.NetworkPolicyIngressRule {
	if app.Spec.AccessPolicy.Ingress.AllowAll {
		return &[]networkingv1.NetworkPolicyIngressRule{{}} // æh funker dette?
	}

	if len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		policies := networkingv1.NetworkPolicyIngressRule{
			From: addNetworkPolicyIngressRules(app),
		}
		return &[]networkingv1.NetworkPolicyIngressRule{policies}
	}

	return &[]networkingv1.NetworkPolicyIngressRule{}
}


func addNetworkPolicyEgressRules(app *nais.Application) (networkPolicy []networkingv1.NetworkPolicyPeer) {
	for _, egress := range app.Spec.AccessPolicy.Egress.Rules {
		networkPolicyPeer := networkingv1.NetworkPolicyPeer{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": egress.Application,
				},
			},
		}

		if egress.Namespace != "" {
			networkPolicyPeer.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": egress.Namespace,
				},
			}
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}

func egressPolicy(app *nais.Application) *[]networkingv1.NetworkPolicyEgressRule {
	if app.Spec.AccessPolicy.Ingress.AllowAll {
		return &[]networkingv1.NetworkPolicyEgressRule{{}} // æh funker dette?
	}

	if len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		policies := networkingv1.NetworkPolicyEgressRule{
			To: addNetworkPolicyEgressRules(app),
		}
		return &[]networkingv1.NetworkPolicyEgressRule{policies}
	}

	return &[]networkingv1.NetworkPolicyEgressRule{}
}

func networkPolicySpec(app *nais.Application) *networkingv1.NetworkPolicySpec {
	ingressPolicy := ingressPolicy(app)
	egressPolicy := egressPolicy(app)

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
		Ingress: *ingressPolicy,
		Egress: *egressPolicy,
	}

}



func NetworkPolicy(app *nais.Application) *networkingv1.NetworkPolicy {
	spec := networkPolicySpec(app)

	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: *spec,
	}
}

