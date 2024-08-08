package securelogs

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
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
						Name: "secure-logs-fluentbit",
					},
					DefaultMode: ptr.To(int32(420)),
				},
			},
		},
		{
			Name: "secure-logs-positiondb",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "secure-logs-buffers",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}
