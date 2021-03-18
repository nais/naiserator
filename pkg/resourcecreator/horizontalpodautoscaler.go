package resourcecreator

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HorizontalPodAutoscaler(app *nais.Application) *v2beta2.HorizontalPodAutoscaler {
	return &v2beta2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: v2beta2.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: v2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2beta2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       app.Name,
			},
			Metrics: []v2beta2.MetricSpec{
				{
					Type: v2beta2.ResourceMetricSourceType,
					Resource: &v2beta2.ResourceMetricSource{
						Name: "cpu",
						Target: v2beta2.MetricTarget{
							Type:               v2beta2.UtilizationMetricType,
							AverageUtilization: int32p(int32(app.Spec.Replicas.CpuThresholdPercentage)),
						},
					},
				},
			},
			MinReplicas: int32p(int32(app.Spec.Replicas.Min)),
			MaxReplicas: int32(app.Spec.Replicas.Max),
		},
	}
}
