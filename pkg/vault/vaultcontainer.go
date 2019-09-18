package vault

import (
	"fmt"
	"strconv"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	config2 "github.com/nais/naiserator/pkg/naiserator/config"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
	k8score "k8s.io/api/core/v1"
)

type config struct {
	vaultAddr          string
	initContainerImage string
	authPath           string
	secretPaths        []nais.SecretPath
	sidecar            bool
}

type initializer struct {
	app    string
	ns     string
	config config
}

// Initializer adds init containers
type Initializer interface {
	AddVaultContainers(podSpec *k8score.PodSpec) k8score.PodSpec
}

func (c config) validate() (bool, error) {

	var result = &multierror.Error{}

	if len(c.vaultAddr) == 0 {
		multierror.Append(result, fmt.Errorf("vault address not found in environment"))
	}

	if len(c.initContainerImage) == 0 {
		multierror.Append(result, fmt.Errorf("vault init container image not found in environment"))
	}

	if len(c.authPath) == 0 {
		multierror.Append(result, fmt.Errorf("vault auth path not found in environment"))
	}

	for _, p := range c.secretPaths {
		if len(p.MountPath) == 0 {
			multierror.Append(result, fmt.Errorf("mount path not specified"))
			break
		}

		if len(p.KvPath) == 0 {
			multierror.Append(result, fmt.Errorf("vault kv path not found in environment"))
			break
		}
	}

	return result.ErrorOrNil() == nil, result.ErrorOrNil()

}

// Enabled checks if this Initializer is enabled
func Enabled() bool {
	return viper.GetBool(config2.FeaturesVault)
}

func DefaultKVPath() string {
	return viper.GetString(config2.VaultKvPath)
}

// NewInitializer creates a new Initializer. Err if required env variables are not set.
func NewInitializer(app *nais.Application) (Initializer, error) {
	config := config{
		vaultAddr:          viper.GetString(config2.VaultAddress),
		initContainerImage: viper.GetString(config2.VaultInitContainerImage),
		authPath:           viper.GetString(config2.VaultAuthPath),
		secretPaths:        app.Spec.Vault.Mounts,
		sidecar:            app.Spec.Vault.Sidecar,
	}

	if ok, err := config.validate(); !ok {
		return nil, err
	}

	return initializer{
		app:    app.Name,
		ns:     app.Namespace,
		config: config,
	}, nil
}

// Add init container to pod spec.
func (c initializer) AddVaultContainers(podSpec *k8score.PodSpec) k8score.PodSpec {
	for index, paths := range c.config.secretPaths {
		volumeName := fmt.Sprintf("vault-secrets-%d", index)
		volume, mount := volumeAndMount(volumeName, paths.MountPath)

		// Add shared volume to pod
		podSpec.Volumes = append(podSpec.Volumes, volume)

		// "Main" container in the pod gets the shared volume mounted.
		mutatedContainers := make([]k8score.Container, 0, len(podSpec.Containers))
		for _, containerCopy := range podSpec.Containers {
			if containerCopy.Name == c.app {
				containerCopy.VolumeMounts = append(containerCopy.VolumeMounts, mount)
			}
			mutatedContainers = append(mutatedContainers, containerCopy)
		}
		podSpec.Containers = mutatedContainers

		// Finally add init container which also gets the shared volume mounted.
		initContainerName := fmt.Sprintf("vks-%d", index)
		podSpec.InitContainers = append(podSpec.InitContainers, c.vaultContainer(initContainerName, mount, paths, false))
		if c.config.sidecar {
			sidecarName := fmt.Sprintf("vks-%d-sidecar", index)
			podSpec.Containers = append(podSpec.Containers, c.vaultContainer(sidecarName, mount, paths, true))
		}
	}

	return *podSpec
}

func volumeAndMount(name, mountPath string) (k8score.Volume, k8score.VolumeMount) {
	volume := k8score.Volume{
		Name: name,
		VolumeSource: k8score.VolumeSource{
			EmptyDir: &k8score.EmptyDirVolumeSource{
				Medium: k8score.StorageMediumMemory,
			},
		},
	}

	mount := k8score.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}

	return volume, mount
}

func (c initializer) vaultRole() string {
	return c.app
}

func (c initializer) vaultContainer(name string, mount k8score.VolumeMount, secretPath nais.SecretPath, isSidecar bool) k8score.Container {
	return k8score.Container{
		Name:         name,
		VolumeMounts: []k8score.VolumeMount{mount},
		Image:        c.config.initContainerImage,
		Env: []k8score.EnvVar{
			{
				Name:  "VKS_VAULT_ADDR",
				Value: c.config.vaultAddr,
			},
			{
				Name:  "VKS_AUTH_PATH",
				Value: c.config.authPath,
			},
			{
				Name:  "VKS_KV_PATH",
				Value: secretPath.KvPath,
			},
			{
				Name:  "VKS_VAULT_ROLE",
				Value: c.vaultRole(),
			},
			{
				Name:  "VKS_SECRET_DEST_PATH",
				Value: secretPath.MountPath,
			},
			{
				Name:  "VKS_IS_SIDECAR",
				Value: strconv.FormatBool(isSidecar),
			},
		},
	}
}
