package poddisruptionbudget

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(source resource.Source, ast *resource.Ast, naisReplicas nais_io_v1.Replicas) {
	if naisReplicas.Max == 1 {
		return
	}

	min := intstr.FromInt(naisReplicas.Min)

	podDisruptionBudget := &policyv1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: "policy/v1beta1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &min,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": source.GetName(),
				},
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, podDisruptionBudget)
}
