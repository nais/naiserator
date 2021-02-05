package resourcecreator_test

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPodDisruptionBudget(t *testing.T) {
	t.Run("normal app should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		pdb := resourcecreator.PodDisruptionBudget(app)

		assert.Equal(t, app.Name, pdb.Name)
		assert.Equal(t, app.Namespace, pdb.Namespace)
		assert.Equal(t, int32(1), pdb.Spec.MinAvailable.IntVal)
		assert.Equal(t, app.Name, pdb.Spec.Selector.MatchLabels["app"])
	})

	t.Run("min = max replicas should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas.Min = app.Spec.Replicas.Max
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		pdb := resourcecreator.PodDisruptionBudget(app)
		assert.Nil(t, pdb)
	})

	t.Run("max replicas = 1 should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas.Max = 1
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		pdb := resourcecreator.PodDisruptionBudget(app)
		assert.Nil(t, pdb)
	})
}
