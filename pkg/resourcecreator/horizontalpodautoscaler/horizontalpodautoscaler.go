package horizontalpodautoscaler

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	resource.Source
	GetReplicas() *nais_io_v1.Replicas
}

func Create(source Source, ast *resource.Ast) {
	replicas := source.GetReplicas()

	if (*replicas.Max) <= 0 {
		return
	}

	hpa := &v2beta2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: v2beta2.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: v2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2beta2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       source.GetName(),
			},
			Metrics: []v2beta2.MetricSpec{
				{
					Type: v2beta2.ResourceMetricSourceType,
					Resource: &v2beta2.ResourceMetricSource{
						Name: "cpu",
						Target: v2beta2.MetricTarget{
							Type:               v2beta2.UtilizationMetricType,
							AverageUtilization: util.Int32p(int32(replicas.CpuThresholdPercentage)),
						},
					},
				},
			},
			MinReplicas: util.Int32p(int32(*replicas.Min)),
			MaxReplicas: int32(*replicas.Max),
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, hpa)
}