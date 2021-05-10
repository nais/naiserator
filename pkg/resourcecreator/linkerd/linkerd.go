package linkerd

import (
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func Create(resourceOptions resource.Options, deployment *appsv1.Deployment) {
	if !resourceOptions.Linkerd {
		return
	}

	podSpec := deployment.Spec.Template.Spec
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, v1.EnvVar{Name: "START_WITHOUT_ENVOY", Value: "true"})

	return
}
