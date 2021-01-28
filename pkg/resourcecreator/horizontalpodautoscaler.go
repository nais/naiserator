package resourcecreator

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HorizontalPodAutoscaler(app *nais.Application) *autoscalingv1.HorizontalPodAutoscaler {
	return &autoscalingv1.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			MinReplicas:                    int32p(int32(app.Spec.Replicas.Min)),
			MaxReplicas:                    int32(app.Spec.Replicas.Max),
			TargetCPUUtilizationPercentage: int32p(int32(app.Spec.Replicas.CpuThresholdPercentage)),
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       app.Name,
			},
		},
	}
}
