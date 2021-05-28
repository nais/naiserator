package securelogs

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)


func secureLogs(resourceOptions resource.Options, podSpec *corev1.PodSpec) *corev1.PodSpec {
	podSpec.Containers = append(podSpec.Containers, FluentdSidecar(resourceOptions))
	podSpec.Containers = append(podSpec.Containers, ConfigmapReloadSidecar(resourceOptions))

	podSpec.Volumes = append(podSpec.Volumes, Volumes()...)

	volumeMount := corev1.VolumeMount{
		Name:      "secure-logs",
		MountPath: "/secure-logs",
	}
	mainContainer := podSpec.Containers[0].DeepCopy()
	mainContainer.VolumeMounts = append(mainContainer.VolumeMounts, volumeMount)
	podSpec.Containers[0] = *mainContainer

	return podSpec
}

func Create(resourceOptions resource.Options, deployment *appsv1.Deployment, naisSecureLogs *nais.SecureLogs) {
	if naisSecureLogs.Enabled {
		podSpec := &deployment.Spec.Template.Spec
		podSpec = secureLogs(resourceOptions, podSpec)
		deployment.Spec.Template.Spec = *podSpec
	}
}
