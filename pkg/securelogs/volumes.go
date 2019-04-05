package securelogs

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Volumes() []corev1.Volume {
	quantity := resource.MustParse("128M")
	return []corev1.Volume{
		{
			Name: "secure-logs",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &quantity,
				},
			},
		},
		{
			Name: "secure-logs-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "secure-logs",
					},
				},
			},
		},
	}
}
