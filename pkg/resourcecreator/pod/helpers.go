package pod

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FromFilesSecretVolume(volumeName, secretName string, items []corev1.KeyToPath) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Items:      items,
			},
		},
	}
}

func WithAdditionalSecret(spec *corev1.PodSpec, secretName, mountPath string) *corev1.PodSpec {
	spec.Volumes = append(spec.Volumes, FromFilesSecretVolume(secretName, secretName, nil))
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
		FromFilesVolumeMount(secretName, "", mountPath))
	return spec
}

func WithAdditionalEnvFromSecret(spec *corev1.PodSpec, secretName string) *corev1.PodSpec {
	spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, EnvFromSecret(secretName))
	return spec
}

func FromFilesVolumeMount(name string, mountPath string, defaultMountPath string) corev1.VolumeMount {
	if len(mountPath) == 0 {
		mountPath = defaultMountPath
	}

	return corev1.VolumeMount{
		Name:      name,
		ReadOnly:  true,
		MountPath: mountPath,
	}
}

func EnvFromSecret(name string) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
}

func ResourceLimits(reqs nais_io_v1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(reqs.Requests.Cpu),
			corev1.ResourceMemory: resource.MustParse(reqs.Requests.Memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(reqs.Limits.Cpu),
			corev1.ResourceMemory: resource.MustParse(reqs.Limits.Memory),
		},
	}
}

func AppClientID(objectMeta metav1.ObjectMeta, cluster string) string {
	return fmt.Sprintf("%s:%s:%s", cluster, objectMeta.Namespace, objectMeta.Name)
}
