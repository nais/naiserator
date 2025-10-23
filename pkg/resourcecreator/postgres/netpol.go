package postgres

import (
	"fmt"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createNetworkPolicies(source Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	createPoolerNetworkPolicy(source, ast, pgClusterName, pgNamespace)
	createSourceNetworkPolicy(source, ast, pgClusterName, pgNamespace)
}

func createPoolerNetworkPolicy(source Source, ast *resource.Ast, pgClusterName string, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.OwnerReferences = nil
	objectMeta.Name = fmt.Sprintf("%s-pooler", pgClusterName)
	objectMeta.Namespace = pgNamespace

	pgNetpol := &v1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: meta_v1.LabelSelector{
				MatchLabels: map[string]string{
					"application":  "db-connection-pooler",
					"cluster-name": pgClusterName,
				},
			},
			Ingress: []v1.NetworkPolicyIngressRule{
				{
					From: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &meta_v1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": source.GetNamespace(),
								},
							},
							PodSelector: &meta_v1.LabelSelector{
								MatchLabels: map[string]string{
									"app": source.GetName(),
								},
							},
						},
					},
				},
			},
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeIngress,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, pgNetpol)
}

func createSourceNetworkPolicy(source Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pg-%s", source.GetName())

	sourceNetpol := &v1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: meta_v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": source.GetName(),
				},
			},
			Egress: []v1.NetworkPolicyEgressRule{
				{
					To: []v1.NetworkPolicyPeer{
						{
							NamespaceSelector: &meta_v1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": pgNamespace,
								},
							},
							PodSelector: &meta_v1.LabelSelector{
								MatchLabels: map[string]string{
									"application":  "db-connection-pooler",
									"cluster-name": pgClusterName,
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
