package linkerd

import (
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

type Config interface {
	IsLinkerdEnabled() bool
}

func Create(ast *resource.Ast, cfg Config) {
	if !cfg.IsLinkerdEnabled() {
		return
	}

	ast.Env = append(ast.Env, corev1.EnvVar{Name: "START_WITHOUT_ENVOY", Value: "true"})
}
