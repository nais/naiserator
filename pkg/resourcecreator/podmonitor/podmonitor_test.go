package podmonitor_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/podmonitor"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	pov1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetPrometheus(t *testing.T) {
	t.Run("Check if no creation when prometheus operator not enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		opts := &generators.Options{}
		app.Spec.Prometheus.Enabled = true
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		podmonitor.Create(app, ast, opts)
		assert.Empty(t, ast.Operations, "No operations should be created")
	})
	t.Run("Check if default values is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		opts := &generators.Options{
			Config: config.Config{
				Features: config.Features{
					PrometheusOperator: true,
				},
			},
		}
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		podmonitor.Create(app, ast, opts)
		assert.Empty(t, ast.Operations, "No operations should be created")
	})

	t.Run("check if correct value is used when enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		opts := &generators.Options{
			Config: config.Config{
				Features: config.Features{
					PrometheusOperator: true,
				},
			},
		}
		app.Spec.Prometheus.Enabled = true
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		podmonitor.Create(app, ast, opts)
		pm := ast.Operations[0].Resource.(*pov1.PodMonitor)
		endpoint := pm.Spec.PodMetricsEndpoints[0]

		assert.Equal(t, "http", endpoint.Port)
		assert.Equal(t, "/metrics", endpoint.Path)
	})

	t.Run("check if correct value is used with other port and path", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		opts := &generators.Options{
			Config: config.Config{
				Features: config.Features{
					PrometheusOperator: true,
				},
			},
		}
		app.Spec.Prometheus.Enabled = true
		app.Spec.Prometheus.Port = "8888"
		app.Spec.Prometheus.Path = "/_metrics"
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		podmonitor.Create(app, ast, opts)
		pm := ast.Operations[0].Resource.(*pov1.PodMonitor)
		endpoint := pm.Spec.PodMetricsEndpoints[0]

		assert.Equal(t, "metrics", endpoint.Port)
		assert.Equal(t, "/_metrics", endpoint.Path)
	})
}
