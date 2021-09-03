package generator

import (
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func GenerateHpa(application v1alpha1.Application) *autoscalingv1.HorizontalPodAutoscaler {
	return &autoscalingv1.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       application.Name,
				APIVersion: "apps/v1",
			},
			MinReplicas:                    pointer.Int32(int32(application.Spec.Replicas.Min)),
			MaxReplicas:                    int32(application.Spec.Replicas.Max),
			TargetCPUUtilizationPercentage: pointer.Int32(int32(application.Spec.Replicas.HpaTargetCPUUtilizationPercentage)),
		},
	}
}
