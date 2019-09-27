package vault_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/spf13/viper"

	"github.com/nais/naiserator/pkg/vault"
	"github.com/sebdah/goldie"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
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

func TestNewInitializer(t *testing.T) {

	var i vault.Initializer
	const appName = "app"

	viper.Reset()
	viper.Set("features.vault", true)
	viper.Set("vault.address", "adr")
	viper.Set("vault.auth-path", "authpath")
	viper.Set("vault.init-container-image", "img")

	t.Run("Initializer mutates podspec correctly", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Name = appName
		app.Namespace = "namespace"
		app.Spec.Vault.Enabled = true

		paths := []nais.SecretPath{
			{
				MountPath: "/first/mount/path",
				KvPath:    "/first/kv/path",
			},
			{
				MountPath: "/second/mount/path",
				KvPath:    "/second/kv/path",
			},
		}
		app.Spec.Vault.Mounts = paths
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		i, err = vault.NewInitializer(app)
		assert.NoError(t, err)

		podSpec := v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: appName,
				},
			},
		}

		mutatedPodSpec := i.AddVaultContainers(&podSpec)

		goldie.AssertJson(t,"default.json", mutatedPodSpec)

	})
}
