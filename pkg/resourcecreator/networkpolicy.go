package resourcecreator

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func egressRule(appName, namespace string) networkingv1.NetworkPolicyEgressRule {
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": namespace,
				},
			},
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
		}},
	}
}

func networkPolicyRules(rules []nais.AccessPolicyRule, options ResourceOptions) (networkPolicy []networkingv1.NetworkPolicyPeer) {
	for _, rule := range rules {

		// non-local access policy rules do not result in network policies
		if !rule.MatchesCluster(options.ClusterName) {
			continue
		}

		networkPolicyPeer := networkingv1.NetworkPolicyPeer{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": rule.Application,
				},
			},
		}

		if rule.Application == "*" {
			networkPolicyPeer = networkingv1.NetworkPolicyPeer{PodSelector: &metav1.LabelSelector{}}
		}

		if rule.Namespace != "" {
			networkPolicyPeer.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": rule.Namespace,
				},
			}
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}

func ingressPolicy(app *nais.Application, options ResourceOptions) []networkingv1.NetworkPolicyIngressRule {
	rules := make([]networkingv1.NetworkPolicyIngressRule, 0)

	rules = append(rules, networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": IstioPrometheusLabelValue,
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": IstioNamespace,
					},
				},
			},
		},
	})

	if len(app.Spec.AccessPolicy.Inbound.Rules) > 0 {
		rules = append(rules, networkingv1.NetworkPolicyIngressRule{
			From: networkPolicyRules(app.Spec.AccessPolicy.Inbound.Rules, options),
		})
	}

	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"istio": IstioIngressGatewayLabelValue,
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"name": IstioNamespace,
						},
					},
				},
			},
		})
	}

	return rules
}

func egressPolicy(app *nais.Application, options ResourceOptions) []networkingv1.NetworkPolicyEgressRule {
	defaultRules := defaultAllowEgress(options.AccessPolicyNotAllowedCIDRs)

	if app.Spec.Tracing != nil && app.Spec.Tracing.Enabled {
		defaultRules = append(defaultRules, egressRule("jaeger", "istio-system"))
	}

	if len(app.Spec.AccessPolicy.Outbound.Rules) > 0 {
		appRules := networkingv1.NetworkPolicyEgressRule{
			To: networkPolicyRules(app.Spec.AccessPolicy.Outbound.Rules, options),
		}
		defaultRules = append(defaultRules, appRules)
	}

	if app.Spec.LeaderElection && len(options.GoogleProjectId) > 0 {
		apiServerAccessRule := networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: options.ApiServerIp,
					},
				},
			},
		}
		defaultRules = append(defaultRules, apiServerAccessRule)
	}

	return defaultRules
}

func networkPolicySpec(app *nais.Application, options ResourceOptions) networkingv1.NetworkPolicySpec {
	return networkingv1.NetworkPolicySpec{
		PodSelector: *labelSelector("app", app.Name),
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: ingressPolicy(app, options),
		Egress:  egressPolicy(app, options),
	}
}

func typeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "NetworkPolicy",
		APIVersion: "networking.k8s.io/v1",
	}
}

func labelSelector(label string, value string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			label: value,
		},
	}
}

func defaultAllowEgress(ipBlockExceptCIDRs []string) []networkingv1.NetworkPolicyEgressRule {
	return []networkingv1.NetworkPolicyEgressRule{
		{
			To: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector:       labelSelector("istio", "istiod"),
					NamespaceSelector: labelSelector("name", IstioNamespace),
				},
				{
					PodSelector:       labelSelector("istio", "ingressgateway"),
					NamespaceSelector: labelSelector("name", IstioNamespace),
				},
				{
					PodSelector: labelSelector("k8s-app", "kube-dns"),
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							// select in all namespaces since labels on kube-system is regularly deleted in GCP
						},
					},
				},
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR:   NetworkPolicyDefaultEgressAllowIPBlock,
						Except: ipBlockExceptCIDRs,
					},
				},
			},
		},
	}
}

func NetworkPolicy(app *nais.Application, options ResourceOptions) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta:   typeMeta(),
		ObjectMeta: app.CreateObjectMeta(),
		Spec:       networkPolicySpec(app, options),
	}
}
