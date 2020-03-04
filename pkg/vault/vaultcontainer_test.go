package vault_test

import (
	"fmt"
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
	viper.Set("vault.auth-path", "auth/kubernetes/preprod/fss/login")
	viper.Set("vault.init-container-image", "navikt/vault-sidekick:v0.3.10-d122b16")

	userSecretsPaths := []nais.SecretPath{
		{
			KvPath:    "/serviceuser/data/test/srvfasit",
			MountPath: "/secrets/credential/srvfasit",
		},
		{
			KvPath:    "/oracle/data/dev/testdb",
			MountPath: "/secrets/oracle/testdb.json",
			Format:    "json",
		},
		{
			KvPath:    "/certificate/data/dev/fasit-keystore",
			MountPath: "/secrets/certificate/fasit-keystore",
		},
	}
	userSecretsSidecarPaths := []nais.SecretPath{
		{
			KvPath:    "/serviceuser/data/test/srvfasit",
			MountPath: "/secrets/credential/srvfasit",
		},
		{
			KvPath:    "/certificate/data/dev/fasit-keystore",
			MountPath: "/secrets/certificate/fasit-keystore",
		},
	}
	defaultSecretsPathOverride := nais.SecretPath{
		KvPath:    "/kv/preprod/fss/fasit/default",
		MountPath: "/var/run/secrets/nais.io/vault",
	}

	tests := []struct {
		name        string
		goldenFile  string
		paths       []nais.SecretPath
		sidecar     bool
	}{
		{"default", "default.json", nil, false},
		{"default with sidecar", "default_sidecar.json", nil, true},
		{"user specified secrets", "user_secrets.json", userSecretsPaths, false},
		{"user specified secrets with default path", "user_secrets_default_path.json", append(userSecretsPaths, defaultSecretsPathOverride), false},
		{"user specified secrets as sidecar", "user_secrets_sidecar.json", userSecretsSidecarPaths, true},
		{"user specified secrets as sidecar with default path", "user_secrets_sidecar_default_path.json", append(userSecretsSidecarPaths, defaultSecretsPathOverride), true},
	}

	for _, loadDefault := range []bool{true, false} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("Initializer mutates podspec correctly %s load defaults %t", test.name, loadDefault), func(t *testing.T) {
				if loadDefault {
					test.goldenFile = fmt.Sprintf("load_defaults/%s", test.goldenFile)
				}

				app.Spec.Vault.Mounts = test.paths
				app.Spec.Vault.Sidecar = test.sidecar
				app.Spec.Vault.LoadDefault = loadDefault

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
			})

		}
	}
}
