package poddisruptionbudget_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/nais/naiserator/pkg/util"
)

func TestPodDisruptionBudget(t *testing.T) {
	t.Run("max replicas = 1 should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.Replicas.Max = util.Intp(1)
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		poddisruptionbudget.Create(app, ast, *app.Spec.Replicas)
		assert.Len(t, ast.Operations, 0)
	})

	t.Run("min replicas = 1 should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ast := resource.NewAst()
		app.Spec.Replicas.Min = util.Intp(1)
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		poddisruptionbudget.Create(app, ast, *app.Spec.Replicas)
		assert.Len(t, ast.Operations, 0)
	})
}
