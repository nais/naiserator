package horizontalpodautoscaler_test

import (
	"fmt"
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/nais/naiserator/pkg/util"
)

func TestHorizontalPodAutoscaler(t *testing.T) {
	t.Run("should not create if min==max replicas", func(t *testing.T) {
		for _, count := range []int{1, 2, 3} {
			app := fixtures.MinimalApplication()
			app.Spec.Replicas = &nais_io_v1.Replicas{
				Min: util.Intp(count),
				Max: util.Intp(count),
			}
			ast := resource.NewAst()
			err := app.ApplyDefaults()
			assert.NoError(t, err)

			horizontalpodautoscaler.Create(app, ast)
			assert.Len(t, ast.Operations, 0)
		}
	})

	t.Run("should not create if max replicas is less than 1", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min: util.Intp(0),
			Max: util.Intp(0),
		}
		ast := resource.NewAst()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		horizontalpodautoscaler.Create(app, ast)
		assert.Len(t, ast.Operations, 0)
	})

	t.Run("should not create with disableautoscaling", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			DisableAutoScaling: true,
		}
		ast := resource.NewAst()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		horizontalpodautoscaler.Create(app, ast)
		assert.Len(t, ast.Operations, 0)
	})

	t.Run("should use value from cpuThresholdPercentage", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                    util.Intp(1),
			Max:                    util.Intp(10),
			CpuThresholdPercentage: 75,
			DisableAutoScaling:     false,
			ScalingStrategy:        nil,
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		operation := ast.Operations[0]
		assert.Equal(t, resource.OperationCreateOrUpdate, operation.Operation)
		hpa, ok := operation.Resource.(*v2.HorizontalPodAutoscaler)
		assert.True(t, ok)
		assert.Equal(t, util.Int32p(int32(app.Spec.Replicas.CpuThresholdPercentage)), hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	})

	t.Run("should use value from scalingStrategy.Cpu.ThresholdPercentage when set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                    util.Intp(1),
			Max:                    util.Intp(10),
			CpuThresholdPercentage: 50,
			DisableAutoScaling:     false,
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				Cpu: &nais_io_v1.CpuScaling{
					ThresholdPercentage: 75,
				},
			},
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		operation := ast.Operations[0]
		assert.Equal(t, resource.OperationCreateOrUpdate, operation.Operation)
		hpa, ok := operation.Resource.(*v2.HorizontalPodAutoscaler)
		assert.True(t, ok)
		assert.Equal(t, util.Int32p(int32(app.Spec.Replicas.ScalingStrategy.Cpu.ThresholdPercentage)), hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	})

	t.Run("should add kafka scale metric when scalingStrategy.Kafka is set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		topic := fmt.Sprintf("%s.mytopic", fixtures.ApplicationTeam)
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                util.Intp(1),
			Max:                util.Intp(10),
			DisableAutoScaling: false,
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				Kafka: &nais_io_v1.KafkaScaling{
					Topic:         topic,
					ConsumerGroup: fixtures.ApplicationName,
					Threshold:     10,
				},
			},
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		operation := ast.Operations[0]
		assert.Equal(t, resource.OperationCreateOrUpdate, operation.Operation)
		hpa, ok := operation.Resource.(*v2.HorizontalPodAutoscaler)
		assert.True(t, ok)

		externalMetric := hpa.Spec.Metrics[0].External
		assert.Equal(t, horizontalpodautoscaler.KafkaConsumerLagMetric, externalMetric.Metric.Name)
		actualAvgValue, _ := externalMetric.Target.AverageValue.AsInt64()
		assert.Equal(t, int64(10), actualAvgValue)
		matchLabels := externalMetric.Metric.Selector.MatchLabels
		assert.Equal(t, topic, matchLabels["topic"])
		assert.Equal(t, fixtures.ApplicationName, matchLabels["group"])
	})

	t.Run("should add both cpu and kafka metric when both are set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		topic := fmt.Sprintf("%s.mytopic", fixtures.ApplicationTeam)
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                util.Intp(1),
			Max:                util.Intp(10),
			DisableAutoScaling: false,
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				Cpu: &nais_io_v1.CpuScaling{
					ThresholdPercentage: 75,
				},
				Kafka: &nais_io_v1.KafkaScaling{
					Topic:         topic,
					ConsumerGroup: fixtures.ApplicationName,
					Threshold:     10,
				},
			},
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		operation := ast.Operations[0]
		assert.Equal(t, resource.OperationCreateOrUpdate, operation.Operation)
		hpa, ok := operation.Resource.(*v2.HorizontalPodAutoscaler)
		assert.True(t, ok)
		assert.Len(t, hpa.Spec.Metrics, 2)
	})
}
