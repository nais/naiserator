package securelogs

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func FluentdSidecar() corev1.Container {
	return corev1.Container{
		Name:            "secure-logs-fluentd",
		Image:           viper.GetString(config.SecurelogsFluentdImage),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("200m"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "secure-logs",
				MountPath: "/secure-logs",
			},
			{
				Name:      "secure-logs-config",
				MountPath: "/fluentd/etc",
				ReadOnly:  true,
			},
			{
				Name:      "ca-bundle-pem",
				MountPath: "/etc/pki/tls/certs/ca-bundle.crt",
				SubPath:   "ca-bundle.pem",
				ReadOnly:  true,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: "NAIS_APP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['app']",
					},
				},
			},
			{
				Name: "NAIS_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "NAIS_TEAM",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['team']",
					},
				},
			},
			{
				Name: "NAIS_HOST",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
		},
	}
}

func ConfigmapReloadSidecar() corev1.Container {
	return corev1.Container{
		Name:            "secure-logs-configmap-reload",
		Image:           viper.GetString(config.SecurelogsConfigMapReloadImage),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			"--volume-dir=/config",
			"--webhook-url=http://localhost:24444/api/config.reload",
			"--webhook-method=GET",
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("50m"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "secure-logs-config",
				MountPath: "/config",
				ReadOnly:  true,
			},
		},
	}
}
