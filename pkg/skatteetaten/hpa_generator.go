package generator

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateHpa(application skatteetaten_no_v1alpha1.Application) *autoscalingv1.HorizontalPodAutoscaler {
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
			MinReplicas:                    util.Int32p(int32(application.Spec.Replicas.Min)),
			MaxReplicas:                    int32(application.Spec.Replicas.Max),
			TargetCPUUtilizationPercentage: util.Int32p(int32(application.Spec.Replicas.HpaTargetCPUUtilizationPercentage)),
		},
	}
}

