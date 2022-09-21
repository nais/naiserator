package horizontalpodautoscaler_test

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"

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
}
