package securelogs

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

func Create(ast *resource.Ast, resourceOptions resource.Options, naisSecureLogs *nais.SecureLogs) {
	if !naisSecureLogs.Enabled {
		return
	}

	ast.Containers = append(ast.Containers, FluentdSidecar(resourceOptions))
	ast.Containers = append(ast.Containers, ConfigmapReloadSidecar(resourceOptions))
	ast.Volumes = append(ast.Volumes, Volumes()...)
	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{
		Name:      "secure-logs",
		MountPath: "/secure-logs",
	})
}
