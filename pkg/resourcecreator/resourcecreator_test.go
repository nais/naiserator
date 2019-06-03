package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/stretchr/testify/assert"
)

// Test that a specified application spec results in the correct Kubernetes resources.
func TestCreate(t *testing.T) {
	t.Run("default application spec merges into empty struct", func(t *testing.T) {
		app := &nais.Application{}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
	})

	t.Run("application spec needs required parameters", func(t *testing.T) {
		app := &nais.Application{}
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})
}
