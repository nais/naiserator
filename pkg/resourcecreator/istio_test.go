package resourcecreator

import (
	"encoding/json"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestIstio(t *testing.T) {
	t.Run("", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		obj, err := ServiceRole(app)

		some, err := json.Marshal(obj)
		assert.NoError(t, err)
		assert.NotEmpty(t, string(some))
	})
}
