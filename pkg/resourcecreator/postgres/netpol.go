package postgres

import (
	"fmt"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"k8s.io/api/networking/v1"
	v2 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNetworkPolicies(source Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	createPostgresNetworkPolicy(source, ast, pgClusterName, pgNamespace)
	createPoolerNetworkPolicy(source, ast, pgClusterName, pgNamespace)
	createSourceNetworkPolicy(source, ast, pgNamespace)
}

func createPostgresNetworkPolicy(source Source, ast *resource.Ast, pgClusterName string, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.OwnerReferences = nil
	objectMeta.Name = pgClusterName
	objectMeta.Namespace = pgNamespace

	pgNetpol := &v1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: v2.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: v2.LabelSelector{
				MatchLabels: map[string]string{
					"application": "spilo",
					"app":         source.GetName(),
				},
			},
			Egress: []v1.NetworkPolicyEgressRule{
				{
					To: []v1.NetworkPolicyPeer{
						{
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"application": "spilo",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
			},
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"application": "spilo",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
				{
					From: []v1.NetworkPolicyPeer{
						{
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"application": "db-connection-pooler",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
				{
					From: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "nais-system",
								},
							},
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": "postgres-operator",
								},
							},
						},
					},
				},
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeEgress,
				v1.PolicyTypeIngress,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, pgNetpol)
}

func createPoolerNetworkPolicy(source Source, ast *resource.Ast, pgClusterName string, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.OwnerReferences = nil
	objectMeta.Name = fmt.Sprintf("%s-pooler", pgClusterName)
	objectMeta.Namespace = pgNamespace

	pgNetpol := &v1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: v2.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: v2.LabelSelector{
				MatchLabels: map[string]string{
					"application": "db-connection-pooler",
					"app":         source.GetName(),
				},
			},
			Egress: []v1.NetworkPolicyEgressRule{
				{
					To: []v1.NetworkPolicyPeer{
						{
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"application": "spilo",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
			},
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "nais-system",
								},
							},
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": "postgres-operator",
								},
							},
						},
					},
				},
				{
					From: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": source.GetNamespace(),
								},
							},
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"app": source.GetName(),
								},
							},
						},
					},
				},
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeEgress,
				v1.PolicyTypeIngress,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, pgNetpol)
}

func createSourceNetworkPolicy(source Source, ast *resource.Ast, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pg-%s", source.GetName())

	sourceNetpol := &v1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: v2.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: v2.LabelSelector{
				MatchLabels: map[string]string{
					"app": source.GetName(),
				},
			},
			Egress: []v1.NetworkPolicyEgressRule{
				{
					To: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": pgNamespace,
								},
							},
							PodSelector: &v2.LabelSelector{
								MatchLabels: map[string]string{
									"application": "db-connection-pooler",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeEgress,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, sourceNetpol)
}
