package generator

import (
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GeneratePodDisruptionBudget(application v1alpha1.Application) *v1beta1.PodDisruptionBudget {

	minAvailable := application.Spec.Replicas.MinAvailable

	if minAvailable == 0 {
		return nil
	}

	podDisruptionBudget := &v1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				IntVal: minAvailable,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: application.StandardLabelSelector(),
			},
		},
	}
	return podDisruptionBudget
}
