package horizontalpodautoscaler

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(app *nais.Application, operations *resource.Operations) {
	hpa := &v2beta2.HorizontalPodAutoscaler{
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
							AverageUtilization: util.Int32p(int32(app.Spec.Replicas.CpuThresholdPercentage)),
						},
					},
				},
			},
			MinReplicas: util.Int32p(int32(app.Spec.Replicas.Min)),
			MaxReplicas: int32(app.Spec.Replicas.Max),
		},
	}

	*operations = append(*operations, resource.Operation{Resource: hpa, Operation: resource.OperationCreateOrUpdate})
}
