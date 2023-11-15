package horizontalpodautoscaler

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	v2 "k8s.io/api/autoscaling/v2"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
)

const (
	KafkaConsumerLagMetric = "kafka_consumergroup_group_lag"
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
	if replicas.DisableAutoScaling || *replicas.Min == *replicas.Max {
		return
	}

	metricSpecs := make([]v2.MetricSpec, 0)

	if replicas.ScalingStrategy != nil {
		if replicas.ScalingStrategy.Cpu != nil {
			metricSpecs = append(metricSpecs, createCpuMetricSpec(replicas.ScalingStrategy.Cpu.ThresholdPercentage))
		}

		if replicas.ScalingStrategy.Kafka != nil {
			metricSpecs = append(metricSpecs, createKafkaMetricSpec(replicas.ScalingStrategy.Kafka))
		}
	} else {
		metricSpecs = append(metricSpecs, createCpuMetricSpec(replicas.CpuThresholdPercentage))
	}

	hpa := &v2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: v2.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       source.GetName(),
			},
			Metrics:     metricSpecs,
			MinReplicas: util.Int32p(int32(*replicas.Min)),
			MaxReplicas: int32(*replicas.Max),
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, hpa)
}

func createKafkaMetricSpec(kafka *nais_io_v1.KafkaScaling) v2.MetricSpec {
	return v2.MetricSpec{
		Type: v2.ExternalMetricSourceType,
		External: &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: KafkaConsumerLagMetric,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"topic": kafka.Topic,
						"group": kafka.ConsumerGroup,
					},
				},
			},
			Target: v2.MetricTarget{
				Type:         v2.AverageValueMetricType,
				AverageValue: k8sresource.NewQuantity(int64(kafka.Threshold), k8sresource.DecimalSI),
			},
		},
	}
}

func createCpuMetricSpec(percentage int) v2.MetricSpec {
	return v2.MetricSpec{
		Type: v2.ResourceMetricSourceType,
		Resource: &v2.ResourceMetricSource{
			Name: "cpu",
			Target: v2.MetricTarget{
				Type:               v2.UtilizationMetricType,
				AverageUtilization: util.Int32p(int32(percentage)),
			},
		},
	}
}
