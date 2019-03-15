package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getDefaultNetworkPolicy creates a NetworkPolicy with default deny all ingress and egress
func getDefaultNetworkPolicy(app *nais.Application) *networking.NetworkPolicy {
	return &networking.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: networking.NetworkPolicySpec{
			PolicyTypes: []networking.PolicyType{
				networking.PolicyTypeIngress,
				networking.PolicyTypeEgress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Ingress: []networking.NetworkPolicyIngressRule{},
			Egress: []networking.NetworkPolicyEgressRule{},
		},
	}
}

func addNetworkPolicyIngressRules(networkPolicy *networking.NetworkPolicy, app *nais.Application) {
	for _, ingress := range app.Spec.AccessPolicy.Ingress.Rules {
		networkPolicyPeer := networking.NetworkPolicyPeer{
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

		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{
				networkPolicyPeer,
			},
		})
	}
}

// allowAll removes a given PolicyType from an NetworkPolicy, which again
// opens up for all traffic of that policy type
func allowAll(allowedPolicyType networking.PolicyType, networkPolicy *networking.NetworkPolicy) {
	policies := networkPolicy.Spec.PolicyTypes
	for index, policyType := range policies {
		if policyType == allowedPolicyType {
			policies[index] = policies[len(policies)-1]
			networkPolicy.Spec.PolicyTypes = policies[:len(policies)-1]
			return
		}
	}
}

func NetworkPolicy(app *nais.Application) (networkPolicy *networking.NetworkPolicy) {
	networkPolicy = getDefaultNetworkPolicy(app)
	if len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		addNetworkPolicyIngressRules(networkPolicy, app)
	}
	if len(app.Spec.AccessPolicy.Egress.Rules) > 0 || app.Spec.AccessPolicy.Egress.AllowAll {
		allowAll(networking.PolicyTypeEgress, networkPolicy)
	}
	if app.Spec.AccessPolicy.Ingress.AllowAll {
		allowAll(networking.PolicyTypeIngress, networkPolicy)
	}

	return
}
