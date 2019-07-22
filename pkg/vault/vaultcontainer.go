package vault

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/spf13/viper"
	k8score "k8s.io/api/core/v1"
	"strconv"
)

const (
	// EnvVaultAddr is the environment name for looking up the address of the Vault server
	EnvVaultAddr = "NAIS_VAULT_ADDR" //
	// EnvInitContainerImage is the environment name for looking up the init container to use
	EnvInitContainerImage = "NAIS_VAULT_INIT_CONTAINER_IMAGE"
	// EnvVaultAuthPath is the environment name for looking up the path to vault kubernetes auth backend
	EnvVaultAuthPath = "NAIS_VAULT_AUTH_PATH"
	// EnvVaultKVPath is the environment name for looking up the path to Vault KV mount
	EnvVaultKVPath = "NAIS_VAULT_KV_PATH"
	// EnvVaultEnabled is the environment name for looking up the enable/disable feature flag
	EnvVaultEnabled = "NAIS_VAULT_ENABLED"
)

type config struct {
	vaultAddr          string
	initContainerImage string
	authPath           string
}

type initializer struct {
	app    string
	ns     string
	config config
}

// Initializer adds init containers
type Initializer interface {
	AddVaultContainers(app *nais.Application, podSpec *k8score.PodSpec) k8score.PodSpec
}

func (c config) validate() (bool, error) {

	var result = &multierror.Error{}

	if len(c.vaultAddr) == 0 {
		multierror.Append(result, fmt.Errorf("vault address not found in environment. Missing %s", EnvVaultAddr))
	}

	if len(c.initContainerImage) == 0 {
		multierror.Append(result, fmt.Errorf("vault address not found in environment. Missing %s", EnvInitContainerImage))
	}

	if len(c.authPath) == 0 {
		multierror.Append(result, fmt.Errorf("auth path not found in environment. Missing %s", EnvVaultAuthPath))
	}

	return result.ErrorOrNil() == nil, result.ErrorOrNil()

}

func init() {
	viper.BindEnv(EnvVaultAddr, EnvVaultAddr)
	viper.BindEnv(EnvInitContainerImage, EnvInitContainerImage)
	viper.BindEnv(EnvVaultAuthPath, EnvVaultAuthPath)
	viper.BindEnv(EnvVaultKVPath, EnvVaultKVPath)

	// temp feature flag. Disable by default
	viper.BindEnv(EnvVaultEnabled, EnvVaultEnabled)
	viper.SetDefault(EnvVaultEnabled, false)

}

// Enabled checks if this Initializer is enabled
func Enabled() bool {
	return viper.GetBool(EnvVaultEnabled)
}

func DefaultKVPath() string {
	return viper.GetString(EnvVaultKVPath)
}

// NewInitializer creates a new Initializer. Err if required env variables are not set.
func NewInitializer(app *nais.Application) (Initializer, error) {
	config := config{
		vaultAddr:          viper.GetString(EnvVaultAddr),
		initContainerImage: viper.GetString(EnvInitContainerImage),
		authPath:           viper.GetString(EnvVaultAuthPath),
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

// Add init/sidecar containers to pod spec.
func (c initializer) AddVaultContainers(app *nais.Application, podSpec *k8score.PodSpec) k8score.PodSpec {
	defaultVolume, defaultMount := volumeAndMount("vault-secrets-default", nais.DefaultVaultMountPath)
	podSpec.Volumes = append(podSpec.Volumes, defaultVolume)
	appendMountToContainer(app.Name, podSpec, defaultMount)

	// Add default init container if not user specified
	if len(app.Spec.Vault.Mounts) == 0 {
		podSpec.InitContainers = append(podSpec.InitContainers, c.vaultContainer("vks-init-default", defaultMount, app.DefaultSecretPath(DefaultKVPath()), false))
	}

	// Add sidecar if specified
	if app.Spec.Vault.Sidecar {
		podSpec.Containers = append(podSpec.Containers, c.vaultContainer("vks-sidecar-default", defaultMount, app.DefaultSecretPath(DefaultKVPath()), true))
	}

	// Add user specified secrets
	for index, paths := range app.Spec.Vault.Mounts {
		volumeName := fmt.Sprintf("vault-secrets-%d", index)
		volume, mount := volumeAndMount(volumeName, paths.MountPath)
		initContainerName := fmt.Sprintf("vks-init-%d", index)
		podSpec.InitContainers = append(podSpec.InitContainers, c.vaultContainer(initContainerName, mount, paths, false))
		podSpec.Volumes = append(podSpec.Volumes, volume)
		appendMountToContainer(app.Name, podSpec, mount)
	}

	return *podSpec
}

func appendMountToContainer(containerName string, podSpec *k8score.PodSpec, volumetMount k8score.VolumeMount) k8score.PodSpec {
	mutatedContainers := make([]k8score.Container, 0, len(podSpec.Containers))
	for _, containerCopy := range podSpec.Containers {
		if containerCopy.Name == containerName {
			containerCopy.VolumeMounts = append(containerCopy.VolumeMounts, volumetMount)
		}
		mutatedContainers = append(mutatedContainers, containerCopy)
	}
	podSpec.Containers = mutatedContainers

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
