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
)

func TestHorizontalPodAutoscaler(t *testing.T) {
	t.Run("should not create if min==max replicas", func(t *testing.T) {
		for _, count := range []int{1, 2, 3} {
			app := fixtures.MinimalApplication()
			app.Spec.Replicas = &nais_io_v1.Replicas{
				Min: new(count),
				Max: new(count),
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
			Min: new(0),
			Max: new(0),
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

	t.Run("should use value from deprecated cpuThresholdPercentage", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                    new(1),
			Max:                    new(10),
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
		//lint:ignore SA1019 deprecated field, but we still support it for backwards compatibility
		assert.Equal(t, new(int32(app.Spec.Replicas.CpuThresholdPercentage)), hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	})

	t.Run("should use value from scalingStrategy.Cpu.ThresholdPercentage when set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                    new(1),
			Max:                    new(10),
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
		assert.Equal(t, new(int32(app.Spec.Replicas.ScalingStrategy.Cpu.ThresholdPercentage)), hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	})

	t.Run("should add kafka scale metric when scalingStrategy.Kafka is set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		topic := fmt.Sprintf("%s.mytopic", fixtures.ApplicationNamespace)
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                new(1),
			Max:                new(10),
			DisableAutoScaling: false,
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				Kafka: &nais_io_v1.KafkaScaling{
					Topic:         topic,
					ConsumerGroup: fixtures.DefaultApplicationName,
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
		assert.Equal(t, fixtures.DefaultApplicationName, matchLabels["group"])
	})

	t.Run("should add both cpu and kafka metric when both are set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		topic := fmt.Sprintf("%s.mytopic", fixtures.ApplicationNamespace)
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:                new(1),
			Max:                new(10),
			DisableAutoScaling: false,
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				Cpu: &nais_io_v1.CpuScaling{
					ThresholdPercentage: 75,
				},
				Kafka: &nais_io_v1.KafkaScaling{
					Topic:         topic,
					ConsumerGroup: fixtures.DefaultApplicationName,
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

	t.Run("scaleUp stabilization window is nil when not set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min:             new(1),
			Max:             new(10),
			ScalingStrategy: nil,
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		hpa := ast.Operations[0].Resource.(*v2.HorizontalPodAutoscaler)
		assert.Nil(t, hpa.Spec.Behavior)
	})

	t.Run("scaleUp stabilization window is nil when explicitly set to 0", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min: new(1),
			Max: new(10),
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				ScaleUpStabilizationWindowSeconds: 0,
			},
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		hpa := ast.Operations[0].Resource.(*v2.HorizontalPodAutoscaler)
		assert.Nil(t, hpa.Spec.Behavior)
	})

	t.Run("scaleUp stabilization window is set from scalingStrategy when configured", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas = &nais_io_v1.Replicas{
			Min: new(1),
			Max: new(10),
			ScalingStrategy: &nais_io_v1.ScalingStrategy{
				Cpu: &nais_io_v1.CpuScaling{
					ThresholdPercentage: 50,
				},
				ScaleUpStabilizationWindowSeconds: 120,
			},
		}
		ast := resource.NewAst()

		horizontalpodautoscaler.Create(app, ast)
		hpa := ast.Operations[0].Resource.(*v2.HorizontalPodAutoscaler)
		assert.NotNil(t, hpa.Spec.Behavior)
		assert.NotNil(t, hpa.Spec.Behavior.ScaleUp)
		assert.Equal(t, new(int32(120)), hpa.Spec.Behavior.ScaleUp.StabilizationWindowSeconds)
	})
}
