package vault

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

type Source interface {
	resource.Source
	GetVault() *nais_io_v1.Vault
}

type Config interface {
	IsVaultEnabled() bool
	GetVaultOptions() config.Vault
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	naisVault := source.GetVault()
	vaultCfg := cfg.GetVaultOptions()

	if !cfg.IsVaultEnabled() || !naisVault.Enabled {
		return nil
	}

	paths := naisVault.Paths
	if paths == nil {
		paths = make([]nais_io_v1.SecretPath, 0)
	}

	for i, p := range paths {
		if len(p.MountPath) == 0 {
			return fmt.Errorf("NAISERATOR-4229: vault config #%d: mount path not specified", i)
		}
	}

	if !defaultPathExists(paths, vaultCfg.KeyValuePath) {
		paths = append(paths, defaultSecretPath(source, vaultCfg.KeyValuePath))
	}

	if err := validateSecretPaths(paths); err != nil {
		return err
	}

	ast.InitContainers = append(ast.InitContainers, createInitContainer(source, vaultCfg, paths))

	if naisVault.Sidecar {
		ast.Containers = append(ast.Containers, createSideCarContainer(vaultCfg))
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
