package securelogs

import (
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

func fluentdSidecar(cfg Config) corev1.Container {
	return corev1.Container{
		Name:            "secure-logs-fluentd",
		Image:           cfg.GetSecureLogsOptions().FluentdImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    k8sResource.MustParse("10m"),
				corev1.ResourceMemory: k8sResource.MustParse("200M"),
			},
		},
		SecurityContext: configureSecurityContext(cfg) ,
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

func configMapReloadSidecar(cfg Config) corev1.Container {
	return corev1.Container{
		Name:            "secure-logs-configmap-reload",
		Image:           cfg.GetSecureLogsOptions().ConfigMapReloadImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			"--volume-dir=/config",
			"--webhook-url=http://localhost:24444/api/config.reload",
			"--webhook-method=GET",
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    k8sResource.MustParse("10m"),
				corev1.ResourceMemory: k8sResource.MustParse("50M"),
			},
		},
		SecurityContext: configureSecurityContext(cfg),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "secure-logs-config",
				MountPath: "/config",
				ReadOnly:  true,
			},
		},
	}
}

func configureSecurityContext(cfg Config) *corev1.SecurityContext {
	ctx := &corev1.SecurityContext{
		RunAsUser:                pointer.Int64(1069),
		RunAsGroup:               pointer.Int64(1069),
		RunAsNonRoot:             pointer.Bool(true),
		Privileged:               pointer.Bool(false),
		AllowPrivilegeEscalation: pointer.Bool(false),
		ReadOnlyRootFilesystem:   pointer.Bool(true),
	}

	if cfg.IsSeccompEnabled() {
		ctx.SeccompProfile = &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		}
	}
	capabilities := &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL"},
	}

	ctx.Capabilities = capabilities
	return ctx
}
