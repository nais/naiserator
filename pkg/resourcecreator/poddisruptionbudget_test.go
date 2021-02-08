package resourcecreator_test

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPodDisruptionBudget(t *testing.T) {
	t.Run("max replicas = 1 should not have pdb", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Replicas.Max = 1
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		pdb := resourcecreator.PodDisruptionBudget(app)
		assert.Nil(t, pdb)
	})
}
