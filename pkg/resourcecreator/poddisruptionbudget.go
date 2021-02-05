package resourcecreator

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func PodDisruptionBudget(app *nais_io_v1alpha1.Application) *policyv1beta1.PodDisruptionBudget {
	if app.Spec.Replicas.Max == 1 {
		return nil
	}

	if app.Spec.Replicas.Min == app.Spec.Replicas.Max {
		return nil
	}

	return &policyv1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionPudget",
			APIVersion: "policy/v1beta1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
		},
	}
}
