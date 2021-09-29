package poddisruptionbudget

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Create(source resource.Source, ast *resource.Ast, naisReplicas nais_io_v1.Replicas) {
	if *naisReplicas.Max == 1 || *naisReplicas.Min == 1 {
		return
	}

	maxUnavailable := intstr.FromInt(1)

	podDisruptionBudget := &policyv1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: "policy/v1beta1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": source.GetName(),
				},
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, podDisruptionBudget)
}
