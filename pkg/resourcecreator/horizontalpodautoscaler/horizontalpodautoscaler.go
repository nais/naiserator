package horizontalpodautoscaler

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(source resource.Source, ast *resource.Ast, naisReplicas nais.Replicas) {
	if !(*naisReplicas.Max > 0) {
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
							AverageUtilization: util.Int32p(int32(naisReplicas.CpuThresholdPercentage)),
						},
					},
				},
			},
			MinReplicas: util.Int32p(int32(*naisReplicas.Min)),
			MaxReplicas: int32(*naisReplicas.Max),
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, hpa)
}


//TODO this is for autoscaling v1 do we want a flag for this or what?
func CreateV1(source resource.Source, ast *resource.Ast, naisReplicas nais.Replicas) {

	if !(*naisReplicas.Max > 0) {
		return
	}

	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       source.GetName(),
				APIVersion: "apps/v1",
			},
			MinReplicas:                    util.Int32p(int32(*naisReplicas.Min)),
			MaxReplicas:                    int32(*naisReplicas.Max),
			TargetCPUUtilizationPercentage: util.Int32p(int32(naisReplicas.CpuThresholdPercentage)),
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, hpa)
}
