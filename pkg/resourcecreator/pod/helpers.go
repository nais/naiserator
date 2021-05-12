package pod

import "k8s.io/api/core/v1"

func FromFilesSecretVolume(volumeName, secretName string, items []v1.KeyToPath) v1.Volume {
	return v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
				Items:      items,
			},
		},
	}
}

func WithAdditionalSecret(spec *v1.PodSpec, secretName, mountPath string) *v1.PodSpec {
	spec.Volumes = append(spec.Volumes, FromFilesSecretVolume(secretName, secretName, nil))
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts,
		FromFilesVolumeMount(secretName, "", mountPath))
	return spec
}

func WithAdditionalEnvFromSecret(spec *v1.PodSpec, secretName string) *v1.PodSpec {
	spec.Containers[0].EnvFrom = append(spec.Containers[0].EnvFrom, envFromSecret(secretName))
	return spec
}

func FromFilesVolumeMount(name string, mountPath string, defaultMountPath string) v1.VolumeMount {
	if len(mountPath) == 0 {
		mountPath = defaultMountPath
	}

	return v1.VolumeMount{
		Name:      name,
		ReadOnly:  true,
		MountPath: mountPath,
	}
}

