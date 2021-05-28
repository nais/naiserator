package poddisruptionbudget

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(objectMeta metav1.ObjectMeta, operations *resource.Operations, naisReplicas nais_io_v1alpha1.Replicas) {
	if naisReplicas.Max == 1 {
		return
	}

	min := intstr.FromInt(naisReplicas.Min)

	podDisruptionBudget := &policyv1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: "policy/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &min,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": objectMeta.Name,
				},
			},
		},
	}

	*operations = append(*operations, resource.Operation{podDisruptionBudget, resource.OperationCreateOrUpdate})
}
