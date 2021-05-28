package vault

import (
	"fmt"
	"path/filepath"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
)

type config struct {
	vaultAddr          string
	initContainerImage string
	authPath           string
	kvPath             string
	objectMeta         metav1.ObjectMeta
	vault              *nais_io_v1alpha1.Vault
}

// Creator creates vault init/sidecar containers
type Creator interface {
	AddVaultContainer(podSpec *corev1.PodSpec) (*corev1.PodSpec, error)
}

func defaultVaultTokenFileName() string {
	return filepath.Join(nais_io_v1alpha1.DefaultVaultMountPath, "vault_token")
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

	for _, p := range c.vault.Paths {
		if len(p.MountPath) == 0 {
			multierror.Append(result, fmt.Errorf("mount path not specified"))
			break
		}

		if len(p.KvPath) == 0 {
			multierror.Append(result, fmt.Errorf("vault kv path not specified"))
			break
		}
	}

	return result.ErrorOrNil() == nil, result.ErrorOrNil()

}

func NewVaultContainerCreator(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, naisVault *nais_io_v1alpha1.Vault) (Creator, error) {
	config := config{
		vaultAddr:          resourceOptions.Vault.Address,
		initContainerImage: resourceOptions.Vault.InitContainerImage,
		authPath:           resourceOptions.Vault.AuthPath,
		kvPath:             resourceOptions.Vault.KeyValuePath,
		objectMeta:         objectMeta,
		vault:              naisVault,
	}

	if ok, err := config.validate(); !ok {
		return nil, err
	}

	return config, nil
}

// AddVaultContainer add init/sidecar containers to pod spec.
func (c config) AddVaultContainer(podSpec *corev1.PodSpec) (*corev1.PodSpec, error) {
	if len(c.vault.Paths) == 0 {
		return c.addVaultContainer(podSpec, []nais_io_v1alpha1.SecretPath{c.defaultSecretPath()})
	} else {
		return c.addVaultContainer(podSpec, c.vault.Paths)
	}
}

func validateSecretPaths(paths []nais_io_v1alpha1.SecretPath) error {
	m := make(map[string]string, len(paths))
	for _, s := range paths {
		if old, exists := m[s.MountPath]; exists {
			return fmt.Errorf("illegal to mount multiple Vault secrets: %s and %s to the same  path: %s", s.KvPath, old, s.MountPath)
		}
		m[s.MountPath] = s.KvPath
	}
	return nil
}

func (c config) addVaultContainer(spec *corev1.PodSpec, paths []nais_io_v1alpha1.SecretPath) (*corev1.PodSpec, error) {

	if err := validateSecretPaths(paths); err != nil {
		return nil, err
	}

	if !c.defaultPathExists(paths) {
		paths = append(paths, c.defaultSecretPath())
	}

	spec.InitContainers = append(spec.InitContainers, c.createInitContainer(paths))

	if c.vault.Sidecar {
		spec.Containers = append(spec.Containers, c.createSideCarContainer())
	}

	spec.Volumes = append(spec.Volumes, corev1.Volume{
		Name: "vault-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	})

	for i := range spec.Containers {
		if spec.Containers[i].Name == c.objectMeta.Name {
			spec.Containers[i].VolumeMounts = append(spec.Containers[i].VolumeMounts, createInitContainerMounts(paths)...)
		}
	}
	return spec, nil
}

func (c config) defaultPathExists(paths []nais_io_v1alpha1.SecretPath) bool {
	for _, path := range paths {
		defaultMountPathExists := filepath.Clean(nais_io_v1alpha1.DefaultVaultMountPath) == filepath.Clean(path.MountPath)
		defaultKvPathExists := filepath.Clean(c.defaultSecretPath().KvPath) == filepath.Clean(path.KvPath)
		if defaultMountPathExists || defaultKvPathExists {
			return true
		}
	}
	return false
}

func (c config) createInitContainer(paths []nais_io_v1alpha1.SecretPath) corev1.Container {
	args := []string{
		"-v=10",
		"-logtostderr",
		"-one-shot",
		fmt.Sprintf("-vault=%s", c.vaultAddr),
		fmt.Sprintf("-save-token=%s", defaultVaultTokenFileName()),
	}

	for _, path := range paths {
		var format string
		if len(path.Format) > 0 {
			format = path.Format
		} else {
			format = "flatten"
		}

		var paramname string
		if format == "flatten" {
			paramname = "dir"
		} else {
			paramname = "file"
		}

		args = append(args, fmt.Sprintf("-cn=secret:%s:%s=%s,fmt=%s,retries=1", path.KvPath, paramname, path.MountPath, format))
	}

	return corev1.Container{
		Name:         "vks-init",
		VolumeMounts: createInitContainerMounts(paths),
		Args:         args,
		Image:        c.initContainerImage,
		Env: []corev1.EnvVar{
			{
				Name:  "VAULT_AUTH_METHOD",
				Value: "kubernetes",
			},
			{
				Name:  "VAULT_SIDEKICK_ROLE",
				Value: c.objectMeta.Name,
			},
			{
				Name:  "VAULT_K8S_LOGIN_PATH",
				Value: c.authPath,
			},
		},
	}
}
func (c config) createSideCarContainer() corev1.Container {
	args := []string{
		"-v=10",
		"-logtostderr",
		"-renew-token",
		fmt.Sprintf("-vault=%s", c.vaultAddr),
	}

	return corev1.Container{
		Name:         "vks-sidecar",
		VolumeMounts: []corev1.VolumeMount{createDefaultMount()},
		Args:         args,
		Image:        c.initContainerImage,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: k8sResource.MustParse("10m"),
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "VAULT_AUTH_METHOD",
				Value: "token",
			},

			{
				Name:  "VAULT_TOKEN_FILE",
				Value: defaultVaultTokenFileName(),
			},
		},
	}

}

func createInitContainerMounts(paths []nais_io_v1alpha1.SecretPath) []corev1.VolumeMount {

	volumeMounts := make([]corev1.VolumeMount, 0, len(paths))
	for _, path := range paths {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "vault-volume",
			MountPath: path.MountPath,
			SubPath:   filepath.Join("vault", path.MountPath), // Just to make sure subpath does not start with "/"
		})
	}

	// Adding default vault mount if it does not exists
	var defaultMountExist = false
	for _, path := range paths {
		if filepath.Clean(nais_io_v1alpha1.DefaultVaultMountPath) == filepath.Clean(path.MountPath) {
			defaultMountExist = true
			break
		}
	}

	if !defaultMountExist {
		volumeMounts = append(volumeMounts, createDefaultMount())
	}
	return volumeMounts
}

func createDefaultMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "vault-volume",
		MountPath: nais_io_v1alpha1.DefaultVaultMountPath,
		SubPath:   filepath.Join("vault", nais_io_v1alpha1.DefaultVaultMountPath), // Just to make sure subpath does not start with "/"
	}
}

func (c config) defaultSecretPath() nais_io_v1alpha1.SecretPath {
	return nais_io_v1alpha1.SecretPath{
		MountPath: nais_io_v1alpha1.DefaultVaultMountPath,
		KvPath:    fmt.Sprintf("%s/%s/%s", c.kvPath, c.objectMeta.Name, c.objectMeta.Namespace),
	}
}
