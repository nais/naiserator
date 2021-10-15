package pod

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
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

func WithAdditionalSecret(ast *resource.Ast, secretName, mountPath string) {
	ast.Volumes = append(ast.Volumes, FromFilesSecretVolume(secretName, secretName, nil))
	ast.VolumeMounts = append(ast.VolumeMounts, FromFilesVolumeMount(secretName, "", mountPath, true))
}

func WithAdditionalEnvFromSecret(ast *resource.Ast, secretName string) {
	ast.EnvFrom = append(ast.EnvFrom, EnvFromSecret(secretName))
}

func FromFilesVolumeMount(name, mountPath, defaultMountPath string, readOnly bool) corev1.VolumeMount {
	if len(mountPath) == 0 {
		mountPath = defaultMountPath
	}

	return corev1.VolumeMount{
		Name:      name,
		ReadOnly:  readOnly,
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

func ResourceLimits(reqs nais_io_v1.ResourceRequirements) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    k8sResource.MustParse(reqs.Requests.Cpu),
			corev1.ResourceMemory: k8sResource.MustParse(reqs.Requests.Memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    k8sResource.MustParse(reqs.Limits.Cpu),
			corev1.ResourceMemory: k8sResource.MustParse(reqs.Limits.Memory),
		},
	}
}

func AppClientID(source resource.Source, cluster string) string {
	return fmt.Sprintf("%s:%s:%s", cluster, source.GetNamespace(), source.GetName())
}

func FromFilesPVCVolume(volumeName, pvcName string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
				ReadOnly:  false,
			},
		},
	}
}
