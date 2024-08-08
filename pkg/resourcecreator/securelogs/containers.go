package securelogs

import (
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func fluentdSidecar(cfg Config) corev1.Container {
	return corev1.Container{
		Name:            "secure-logs-fluentbit",
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
		Image:           cfg.GetSecureLogsOptions().LogShipperImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/fluent-bit/bin/fluent-bit",
			"-c",
			"/fluent-bit/etc-operator/fluent-bit.conf",
		},
		Env: []corev1.EnvVar{
			{
				Name: "NAIS_NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
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
				Name: "NAIS_APP_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.labels['app']",
					},
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    k8sResource.MustParse("10m"),
				corev1.ResourceMemory: k8sResource.MustParse("15M"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: k8sResource.MustParse("30M"),
			},
		},

		SecurityContext: &corev1.SecurityContext{
			Privileged:               ptr.To(false),
			AllowPrivilegeEscalation: ptr.To(false),
			ReadOnlyRootFilesystem:   ptr.To(true),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			RunAsUser:    ptr.To(int64(1065)),
			RunAsGroup:   ptr.To(int64(1065)),
			RunAsNonRoot: ptr.To(true),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "secure-logs",
				MountPath: "/secure-logs",
			},
			{
				Name:      "secure-logs-config",
				MountPath: "/fluent-bit/etc-operator",
				ReadOnly:  true,
			},
			{
				Name:      "secure-logs-positiondb",
				MountPath: "/tail-db",
			},
			{
				Name:      "secure-logs-buffers",
				MountPath: "/buffers",
			},
		},
	}
}
