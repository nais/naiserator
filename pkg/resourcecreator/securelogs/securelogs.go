package securelogs

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

type Source interface {
	resource.Source
	GetSecureLogs() *nais_io_v1.SecureLogs
}

type Config interface {
	GetSecureLogsOptions() config.Securelogs
}

func Create(source Source, ast *resource.Ast, cfg Config) {
	seclog := source.GetSecureLogs()
	if seclog == nil || !seclog.Enabled {
		return
	}

	opts := cfg.GetSecureLogsOptions()

	ast.Containers = append(ast.Containers, fluentdSidecar(opts.FluentdImage))
	ast.Containers = append(ast.Containers, configMapReloadSidecar(opts.ConfigMapReloadImage))
	ast.Volumes = append(ast.Volumes, Volumes()...)
	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{
		Name:      "secure-logs",
		MountPath: "/secure-logs",
	})
}
