package vault

import (
	"os"
	"testing"

	"github.com/nais/naisd/pkg/test"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

var envVars = map[string]string{
	EnvVaultAuthPath:      "authpath",
	EnvVaultKVPath:        "kvpath",
	EnvInitContainerImage: "image",
	EnvVaultAddr:          "adr",
}

func TestFeatureFlagging(t *testing.T) {
	t.Run("Vault should by default be disabled", func(t *testing.T) {
		assert.False(t, Enabled())
	})

	t.Run("Feature flag is configured through env variables", func(t *testing.T) {
		os.Setenv(EnvVaultEnabled, "true")

		assert.True(t, Enabled())

		os.Unsetenv(EnvVaultEnabled)

	})
}

func TestConfigValidation(t *testing.T) {

	t.Run("Validation should fail if one or more config value is missing", func(t *testing.T) {

		var tests = []struct {
			config         config
			expectedResult bool
		}{
			{config{vaultAddr: "addr"}, false},
			{config{vaultAddr: "addr", kvPath: "path"}, false},
			{config{vaultAddr: "addr", kvPath: "path", authPath: "auth"}, false},
		}

		for _, testCase := range tests {
			actualResult, err := testCase.config.validate()
			assert.Equal(t, testCase.expectedResult, actualResult)
			assert.NotNil(t, err)
		}
	})

	t.Run("Validation should pass if all config values are present", func(t *testing.T) {
		result, err := config{vaultAddr: "addr", kvPath: "path", authPath: "auth", initContainerImage: "image"}.validate()
		assert.True(t, result)
		assert.Nil(t, err)

	})
}

func TestNewInitializer(t *testing.T) {

	t.Run("Initializer is configured through environment variables", test.EnvWrapper(envVars, func(t *testing.T) {

		aInitializer, e := NewInitializer("app", "ns")
		assert.NoError(t, e)
		assert.NotNil(t, aInitializer)

		initializerStruct, ok := aInitializer.(initializer)
		assert.True(t, ok)

		config := initializerStruct.config
		assert.NotNil(t, config)
		assert.Equal(t, envVars[EnvVaultAddr], config.vaultAddr)
		assert.Equal(t, envVars[EnvInitContainerImage], config.initContainerImage)
		assert.Equal(t, envVars[EnvVaultKVPath], config.kvPath)
		assert.Equal(t, envVars[EnvVaultAuthPath], config.authPath)

	}))

	t.Run("Fail initializer creation if config validation fails", func(t *testing.T) {
		_, err := NewInitializer("app", "ns")
		assert.Error(t, err)
	})
}

func TestInitializer_AddInitContainer(t *testing.T) {
	config := config{authPath: "authpath", kvPath: "kvpath", initContainerImage: "image", vaultAddr: "http://localhost"}

	initializer := initializer{app: "app", ns: "env", config: config}
	expectedVolume, expectedMount := volumeAndMount()
	expectedInitContainer := initializer.initContainer(expectedMount)

	t.Run("Add init container to pod spec", func(t *testing.T) {
		podSpec := &v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: initializer.app,
				},
			},
		}
		actualPodSpec := initializer.AddInitContainer(podSpec)

		assert.Equal(t, 1, len(actualPodSpec.InitContainers))
		assert.Equal(t, expectedInitContainer, actualPodSpec.InitContainers[0])
		assert.Equal(t, 1, len(actualPodSpec.Volumes))
		assert.Equal(t, expectedVolume, actualPodSpec.Volumes[0])

		assert.Equal(t, 1, len(actualPodSpec.Containers))
		assert.Equal(t, 1, len(actualPodSpec.Containers[0].VolumeMounts))
		assert.Equal(t, expectedMount, actualPodSpec.Containers[0].VolumeMounts[0])

	})

	t.Run("Secrets are not mounted into sidecar containers", func(t *testing.T) {
		podSpec := &v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: initializer.app,
				},
				{
					Name: "sidecar",
				},
			},
		}

		actualPodSpec := initializer.AddInitContainer(podSpec)

		assert.Equal(t, 2, len(actualPodSpec.Containers))

		for _, container := range podSpec.Containers {
			if container.Name == initializer.app {
				assert.NotEmpty(t, container.VolumeMounts)
			} else {
				assert.Empty(t, container.VolumeMounts)
			}
		}
	})
}

func TestVolumeAndMountCreation(t *testing.T) {
	volume, mount := volumeAndMount()

	assert.Equal(t, volume.Name, mount.Name)
	assert.NotEmpty(t, volume.EmptyDir)
	assert.Equal(t, v1.StorageMediumMemory, volume.EmptyDir.Medium)
	assert.Equal(t, mountPath, mount.MountPath)
}

func TestInitContainerCreation(t *testing.T) {
	config := config{authPath: "authpath", kvPath: "kvpath", initContainerImage: "image", vaultAddr: "http://localhost"}

	initializer := initializer{app: "app", ns: "env", config: config}
	_, expectedMount := volumeAndMount()
	actualContainer := initializer.initContainer(expectedMount)

	assert.Equal(t, 1, len(actualContainer.VolumeMounts))
	assert.Equal(t, expectedMount, actualContainer.VolumeMounts[0])
	assert.Equal(t, config.initContainerImage, actualContainer.Image)
	assert.Equal(t, 5, len(actualContainer.Env))

	for _, envVar := range actualContainer.Env {
		switch envVar.Name {
		case "VKS_SECRET_DEST_PATH":
			assert.Equal(t, mountPath, envVar.Value)
		case "VKS_VAULT_ADDR":
			assert.Equal(t, config.vaultAddr, envVar.Value)
		case "VKS_AUTH_PATH":
			assert.Equal(t, config.authPath, envVar.Value)
		case "VKS_KV_PATH":
			assert.Equal(t, initializer.kvPath(), envVar.Value)
		case "VKS_VAULT_ROLE":
			assert.Equal(t, initializer.vaultRole(), envVar.Value)
		default:
			t.Errorf("Illegal envvar %s", envVar)
		}
	}
}

func TestKVPath(t *testing.T) {
	initializer := initializer{
		config: config{
			kvPath: "path/kvpath",
		},
		app: "app",
		ns:  "env",
	}
	assert.Equal(t, "path/kvpath/app/env", initializer.kvPath())
}

func TestRole(t *testing.T) {
	initializer := initializer{
		app: "app",
	}
	assert.Equal(t, "app", initializer.vaultRole())
}
