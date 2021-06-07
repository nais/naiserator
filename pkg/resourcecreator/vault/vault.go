package vault

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

func Create(source resource.Source, ast *resource.Ast, options resource.Options, naisVault *nais_io_v1.Vault) error {
	if !options.VaultEnabled || !naisVault.Enabled {
		return nil
	}

	paths := naisVault.Paths
	if paths == nil {
		paths = make([]nais_io_v1.SecretPath, 0)
	}

	for i, p := range paths {
		if len(p.MountPath) == 0 {
			return fmt.Errorf("vault config #%d: mount path not specified", i)
		}
	}

	if !defaultPathExists(paths, options.Vault.KeyValuePath) {
		paths = append(paths, defaultSecretPath(source, options.Vault.KeyValuePath))
	}

	if err := validateSecretPaths(paths); err != nil {
		return err
	}

	ast.InitContainers = append(ast.InitContainers, createInitContainer(source, options, paths))

	if naisVault.Sidecar {
		ast.Containers = append(ast.Containers, createSideCarContainer(options))
	}

	ast.Volumes = append(ast.Volumes, corev1.Volume{
		Name: "vault-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	})

	ast.VolumeMounts = append(ast.VolumeMounts, createInitContainerMounts(paths)...)

	return nil
}
