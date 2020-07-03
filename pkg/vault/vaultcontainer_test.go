package vault_test

import (
	"github.com/nais/naiserator/pkg/vault"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFeatureFlagging(t *testing.T) {
	t.Run("Vault should by default be disabled", func(t *testing.T) {
		viper.Reset()
		assert.False(t, vault.Enabled())
	})

	t.Run("Feature flag is configured through env variables", func(t *testing.T) {
		viper.Set("features.vault", true)
		assert.True(t, vault.Enabled())
	})
}
