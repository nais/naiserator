package poddisruptionbudget_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestPodDisruptionBudget(t *testing.T) {
	t.Run("max replicas = 1 should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ops := resource.Operations{}
		app.Spec.Replicas.Max = 1
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		poddisruptionbudget.Create(app, &ops)
		assert.Len(t, ops, 0)
	})
}
