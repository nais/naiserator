package vault_test

import (
	"os"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/nais/naiserator/pkg/test"
	"github.com/nais/naiserator/pkg/vault"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

var envVars = map[string]string{
	vault.EnvVaultAuthPath:      "authpath",
	vault.EnvInitContainerImage: "image",
	vault.EnvVaultAddr:          "adr",
}

func TestFeatureFlagging(t *testing.T) {
	t.Run("Vault should by default be disabled", func(t *testing.T) {
		assert.False(t, vault.Enabled())
	})

	t.Run("Feature flag is configured through env variables", func(t *testing.T) {
		os.Setenv(vault.EnvVaultEnabled, "true")

		assert.True(t, vault.Enabled())

		os.Unsetenv(vault.EnvVaultEnabled)

	})
}

func TestNewInitializer(t *testing.T) {

	const appName = "app"

	setup := func(config nais.Vault) (*nais.Application, vault.Initializer) {
		app := fixtures.MinimalApplication()
		app.Name = appName
		app.Namespace = "namespace"
		app.Spec.Vault = config

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		initializer, err := vault.NewInitializer(app)
		assert.NoError(t, err)

		return app, initializer
	}

	podSpec := func() *v1.PodSpec {
		return &v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: appName,
				},
			},
		}
	}

	t.Run("Initializer adds default containers correctly", test.EnvWrapper(envVars, func(t *testing.T) {
		app, i := setup(nais.Vault{Enabled: true})

		mutatedPodSpec := i.AddVaultContainers(app, podSpec())

		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.Volumes[0].Name)

		//App container
		assert.Len(t, mutatedPodSpec.Containers, 1)
		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.Containers[0].VolumeMounts[0].Name)
		assert.Equal(t, nais.DefaultVaultMountPath, mutatedPodSpec.Containers[0].VolumeMounts[0].MountPath)
		//InitContainer
		assert.Len(t, mutatedPodSpec.InitContainers, 1)
		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.InitContainers[0].VolumeMounts[0].Name)
		assert.Equal(t, nais.DefaultVaultMountPath, mutatedPodSpec.InitContainers[0].VolumeMounts[0].MountPath)

	}))

	t.Run("Initializer adds sidecar container correctly", test.EnvWrapper(envVars, func(t *testing.T) {
		app, i := setup(nais.Vault{Enabled: true, Sidecar: true})

		mutatedPodSpec := i.AddVaultContainers(app, podSpec())

		assert.Len(t, mutatedPodSpec.Containers, 2)
		//App container
		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.Containers[0].VolumeMounts[0].Name)
		assert.Equal(t, nais.DefaultVaultMountPath, mutatedPodSpec.Containers[0].VolumeMounts[0].MountPath)
		//Sidecar
		assert.Equal(t, "vks-sidecar-default", mutatedPodSpec.Containers[1].Name)
		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.Containers[1].VolumeMounts[0].Name)
		assert.Equal(t, nais.DefaultVaultMountPath, mutatedPodSpec.Containers[0].VolumeMounts[0].MountPath)
		//InitContainer
		assert.Len(t, mutatedPodSpec.InitContainers, 1)
		assert.Equal(t, "vks-init-default", mutatedPodSpec.InitContainers[0].Name)
		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.InitContainers[0].VolumeMounts[0].Name)
		assert.Equal(t, nais.DefaultVaultMountPath, mutatedPodSpec.InitContainers[0].VolumeMounts[0].MountPath)

	}))

	t.Run("Initializer mutates podspec correctly", test.EnvWrapper(envVars, func(t *testing.T) {

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

		app, i := setup(nais.Vault{Enabled: true, Sidecar: true, Mounts: paths})

		mutatedPodSpec := i.AddVaultContainers(app, podSpec())

		// Verify that the correct number of objects have been created
		assert.Len(t, mutatedPodSpec.Containers, 2)
		assert.Len(t, mutatedPodSpec.InitContainers, 2)
		assert.Len(t, mutatedPodSpec.Volumes, 3)

		// Verify unique names on all volumes and mounts
		assert.Equal(t, "vault-secrets-default", mutatedPodSpec.Volumes[0].Name)
		assert.Equal(t, "vault-secrets-0", mutatedPodSpec.Volumes[1].Name)
		assert.Equal(t, "vault-secrets-1", mutatedPodSpec.Volumes[2].Name)
		assert.Equal(t, "vks-init-0", mutatedPodSpec.InitContainers[0].Name)
		assert.Equal(t, "vks-init-1", mutatedPodSpec.InitContainers[1].Name)

		// Verify that the main container has three vault secret paths mounted
		assert.Equal(t, nais.DefaultVaultMountPath, mutatedPodSpec.Containers[0].VolumeMounts[0].MountPath)
		assert.Equal(t, "/first/mount/path", mutatedPodSpec.Containers[0].VolumeMounts[1].MountPath)
		assert.Equal(t, "/second/mount/path", mutatedPodSpec.Containers[0].VolumeMounts[2].MountPath)
		assert.Equal(t, v1.StorageMedium("Memory"), mutatedPodSpec.Volumes[0].EmptyDir.Medium)
		assert.Equal(t, v1.StorageMedium("Memory"), mutatedPodSpec.Volumes[1].EmptyDir.Medium)

		// Verify that both vault init container has correct configuration
		for i := range paths {
			assert.Equal(t, envVars[vault.EnvInitContainerImage], mutatedPodSpec.InitContainers[i].Image)
			assert.Equal(t, envVars[vault.EnvVaultAddr], test.EnvVar(mutatedPodSpec.InitContainers[i].Env, "VKS_VAULT_ADDR"))
			assert.Equal(t, envVars[vault.EnvVaultAuthPath], test.EnvVar(mutatedPodSpec.InitContainers[i].Env, "VKS_AUTH_PATH"))
			assert.Equal(t, appName, test.EnvVar(mutatedPodSpec.InitContainers[i].Env, "VKS_VAULT_ROLE"))
			assert.Equal(t, paths[i].KvPath, test.EnvVar(mutatedPodSpec.InitContainers[i].Env, "VKS_KV_PATH"))
			assert.Equal(t, paths[i].MountPath, test.EnvVar(mutatedPodSpec.InitContainers[i].Env, "VKS_SECRET_DEST_PATH"))
			assert.Equal(t, paths[i].MountPath, mutatedPodSpec.InitContainers[i].VolumeMounts[0].MountPath)
		}

	}))
}
