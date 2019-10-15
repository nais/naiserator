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

func TestVaultContainerCreation(t *testing.T) {

	var i vault.Creator
	const appName = "fasit"

	app := fixtures.MinimalApplication()
	app.Name = appName
	app.Namespace = "default"
	app.Spec.Vault.Enabled = true

	viper.Reset()
	viper.Set("features.vault", true)
	viper.Set("vault.address", "https://vault.adeo.no")
	viper.Set("vault.kv-path", "/kv/preprod/fss")
	viper.Set("vault.auth-path-new", "auth/kubernetes/preprod/fss/login") //todo Breaking change with nais-yaml
	viper.Set("vault.init-container-image", "navikt/vault-sidekick:v0.3.10-d122b16")

	t.Run("Initializer mutates podspec correctly", func(t *testing.T) {
		tests := []struct {
			name       string
			goldenFile string
			paths      []nais.SecretPath
			sidcar     bool
		}{
			{"default", "default.json", nil, false},
			{"default with sidecaar", "default_sidecar.json", nil, true},
			{"user specified secrets", "user_secrets.json", []nais.SecretPath{
				{
					KvPath:    "/serviceuser/data/test/srvfasit",
					MountPath: "/secrets/credential/srvfasit",
				},
				{
					KvPath:    "/certificate/data/dev/fasit-keystore",
					MountPath: "/secrets/certificate/fasit-keystore",
				},
			}, false},
			{"user specified secrets as sidecar ", "user_secrets_sidecar.json", []nais.SecretPath{
				{
					KvPath:    "/serviceuser/data/test/srvfasit",
					MountPath: "/secrets/credential/srvfasit",
				},
				{
					KvPath:    "/certificate/data/dev/fasit-keystore",
					MountPath: "/secrets/certificate/fasit-keystore",
				},
			}, true},
		}

		for _, test := range tests {
			app.Spec.Vault.Mounts = test.paths
			app.Spec.Vault.Sidecar = test.sidcar

			err := nais.ApplyDefaults(app)
			assert.NoError(t, err)

			i, err = vault.NewVaultContainerCreator(*app)
			assert.NoError(t, err)

			podSpec := v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  appName,
						Image: "oyvindio/debug",
					},
				},
			}

			mutatedPodSpec, err := i.AddVaultContainer(&podSpec)
			assert.NoError(t, err)

			goldie.AssertJson(t, test.goldenFile, mutatedPodSpec)
		}

	})
}
