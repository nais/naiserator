package resourcecreator

import (
	"encoding/json"
	"testing"

	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestIstio(t *testing.T) {
	t.Run("", func(t *testing.T) {
		app := fixtures.Application()
		obj := Istio(app)

		some, err := json.Marshal(obj)
		assert.NoError(t, err)
		assert.NotEmpty(t, string(some))
	})
}
