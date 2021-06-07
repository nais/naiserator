package vault

import (
	"fmt"
	"path/filepath"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
)

func defaultVaultTokenFileName() string {
	return filepath.Join(nais_io_v1.DefaultVaultMountPath, "vault_token")
}

func validateSecretPaths(paths []nais_io_v1.SecretPath) error {
	m := make(map[string]string, len(paths))
	for _, s := range paths {
		if old, exists := m[s.MountPath]; exists {
			return fmt.Errorf("illegal to mount multiple Vault secrets: %s and %s to the same path: %s", s.KvPath, old, s.MountPath)
		}
		m[s.MountPath] = s.KvPath
	}
	return nil
}

func defaultPathExists(paths []nais_io_v1.SecretPath, kvPath string) bool {
	kvPath = filepath.Clean(kvPath)
	for _, path := range paths {
		defaultMountPathExists := filepath.Clean(nais_io_v1.DefaultVaultMountPath) == filepath.Clean(path.MountPath)
		defaultKvPathExists := kvPath == filepath.Clean(path.KvPath)
		if defaultMountPathExists || defaultKvPathExists {
			return true
		}
	}
	return false
}

func createInitContainer(source resource.Source, options resource.Options, paths []nais_io_v1.SecretPath) corev1.Container {
	args := []string{
		"-v=10",
		"-logtostderr",
		"-one-shot",
		fmt.Sprintf("-vault=%s", options.Vault.Address),
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
		Image:        options.Vault.InitContainerImage,
		Env: []corev1.EnvVar{
			{
				Name:  "VAULT_AUTH_METHOD",
				Value: "kubernetes",
			},
			{
				Name:  "VAULT_SIDEKICK_ROLE",
				Value: source.GetName(),
			},
			{
				Name:  "VAULT_K8S_LOGIN_PATH",
				Value: options.Vault.AuthPath,
			},
		},
	}
}
func createSideCarContainer(options resource.Options) corev1.Container {
	args := []string{
		"-v=10",
		"-logtostderr",
		"-renew-token",
		fmt.Sprintf("-vault=%s", options.Vault.Address),
	}

	return corev1.Container{
		Name:         "vks-sidecar",
		VolumeMounts: []corev1.VolumeMount{createDefaultMount()},
		Args:         args,
		Image:        options.Vault.InitContainerImage,
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

func createInitContainerMounts(paths []nais_io_v1.SecretPath) []corev1.VolumeMount {

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
		if filepath.Clean(nais_io_v1.DefaultVaultMountPath) == filepath.Clean(path.MountPath) {
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
		MountPath: nais_io_v1.DefaultVaultMountPath,
		SubPath:   filepath.Join("vault", nais_io_v1.DefaultVaultMountPath), // Just to make sure subpath does not start with "/"
	}
}

func defaultSecretPath(source resource.Source, kvPath string) nais_io_v1.SecretPath {
	return nais_io_v1.SecretPath{
		MountPath: nais_io_v1.DefaultVaultMountPath,
		KvPath:    fmt.Sprintf("%s/%s/%s", kvPath, source.GetName(), source.GetNamespace()),
	}
}
