package vault

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/viper"
	k8score "k8s.io/api/core/v1"
)

const (
	mountPath = "/var/run/secrets/naisd.io/vault"
	// EnvVaultAddr is the environment name for looking up the address of the Vault server
	EnvVaultAddr = "NAISD_VAULT_ADDR" //
	// EnvInitContainerImage is the environment name for looking up the init container to use
	EnvInitContainerImage = "NAISD_VAULT_INIT_CONTAINER_IMAGE"
	// EnvVaultAuthPath is the environment name for looking up the path to vault kubernetes auth backend
	EnvVaultAuthPath = "NAISD_VAULT_AUTH_PATH"
	// EnvVaultKVPath is the environment name for looking up the path to Vault KV mount
	EnvVaultKVPath = "NAISD_VAULT_KV_PATH"
	// EnvVaultEnabled is the environment name for looking up the enable/disable feature flag
	EnvVaultEnabled = "NAISD_VAULT_ENABLED"
)

type config struct {
	vaultAddr          string
	initContainerImage string
	authPath           string
	kvPath             string
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

	if len(c.kvPath) == 0 {
		multierror.Append(result, fmt.Errorf("kv path not found in environment. Missing %s", EnvVaultKVPath))
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

type initializer struct {
	app    string
	ns     string
	config config
}

// Initializer adds init containers
type Initializer interface {
	AddInitContainer(podSpec *k8score.PodSpec) k8score.PodSpec
}

// Enabled checks if this Initalizer is enabled
func Enabled() bool {
	return viper.GetBool(EnvVaultEnabled)
}

// NewInitializer creates a new Initializer. Err if required env variables are not set.
func NewInitializer(app, ns string) (Initializer, error) {
	config := config{
		vaultAddr:          viper.GetString(EnvVaultAddr),
		initContainerImage: viper.GetString(EnvInitContainerImage),
		authPath:           viper.GetString(EnvVaultAuthPath),
		kvPath:             viper.GetString(EnvVaultKVPath),
	}

	if ok, err := config.validate(); !ok {
		return nil, err
	}

	return initializer{
		app:    app,
		ns:     ns,
		config: config,
	}, nil
}

// Add init container to pod spec.
func (c initializer) AddInitContainer(podSpec *k8score.PodSpec) k8score.PodSpec {
	volume, mount := volumeAndMount()

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
	podSpec.InitContainers = append(podSpec.InitContainers, c.initContainer(mount))

	return *podSpec
}

func volumeAndMount() (k8score.Volume, k8score.VolumeMount) {
	name := "vault-secrets"
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

func (c initializer) kvPath() string {
	return c.config.kvPath + "/" + c.app + "/" + c.ns
}

func (c initializer) vaultRole() string {
	return c.app
}

func (c initializer) initContainer(mount k8score.VolumeMount) k8score.Container {
	return k8score.Container{
		Name:         "vks",
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
				Value: c.kvPath(),
			},
			{
				Name:  "VKS_VAULT_ROLE",
				Value: c.vaultRole(),
			},
			{
				Name:  "VKS_SECRET_DEST_PATH",
				Value: mountPath,
			},
		},
	}

}
