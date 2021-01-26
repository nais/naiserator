package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestServiceEntry(t *testing.T) {
	t.Run("serviceentry not created if external hosts array omitted", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		serviceEntries, err := resourcecreator.ServiceEntries(app)
		assert.NoError(t, err)
		assert.Len(t, serviceEntries, 0)
	})
}
