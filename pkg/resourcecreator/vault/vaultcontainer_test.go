package vault_test

import (
	"testing"

	vault2 "github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestFeatureFlagging(t *testing.T) {
	t.Run("Vault should by default be disabled", func(t *testing.T) {
		viper.Reset()
		assert.False(t, vault2.Enabled())
	})

	t.Run("Feature flag is configured through env variables", func(t *testing.T) {
		viper.Set("features.vault", true)
		assert.True(t, vault2.Enabled())
	})
}
