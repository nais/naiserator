package cnpg

import (
	"fmt"

	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func createNetworkPolicies(source resource.Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	createPoolerNetworkPolicy(source, ast, pgClusterName, pgNamespace)
	createSourceNetworkPolicy(source, ast, pgClusterName, pgNamespace)
}

func createPoolerNetworkPolicy(source resource.Source, ast *resource.Ast, pgClusterName string, pgNamespace string) {
	var err error
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name, err = namegen.SuffixedShortName(fmt.Sprintf("%s-%s", source.GetName(), pgClusterName), "ingress", validation.DNS1123SubdomainMaxLength)
	if err != nil {
		panic(fmt.Sprintf("Error when hashing, this should be impossible: %v", err))
	}
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
					"cnpg.io/podRole": "pooler",
					"cnpg.io/cluster": pgClusterName,
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

func createSourceNetworkPolicy(source resource.Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	var err error
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name, err = namegen.SuffixedShortName(fmt.Sprintf("%s-%s", source.GetName(), pgClusterName), "egress", validation.DNS1123SubdomainMaxLength)
	if err != nil {
		panic(fmt.Sprintf("Error when hashing, this should be impossible: %v", err))
	}

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
									"cnpg.io/podRole": "pooler",
									"cnpg.io/cluster": pgClusterName,
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
