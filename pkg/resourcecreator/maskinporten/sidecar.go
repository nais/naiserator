package maskinporten

import (
	"strings"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func texasSidecar(cfg Config, secretNames, providers []string) corev1.Container {
	envFroms := make([]corev1.EnvFromSource, 0, len(secretNames))
	for _, secretName := range secretNames {
		envFroms = append(envFroms, pod.EnvFromSecret(secretName))
	}
	envs := []corev1.EnvVar{
		{
			Name:  "BIND_ADDRESS",
			Value: "127.0.0.1:1337",
		},
	}
	for _, provider := range providers {
		envs = append(envs, corev1.EnvVar{
			Name:  strings.ToUpper(provider) + "_ENABLED",
			Value: "true",
		})
	}

	return corev1.Container{
		Name:            "texas",
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
		Image:           cfg.GetTexasOptions().Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envs,
		EnvFrom:         envFroms,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    k8sResource.MustParse("20m"),
				corev1.ResourceMemory: k8sResource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: k8sResource.MustParse("256Mi"),
			},
		},

		// FIXME: duplicated in securelogs/containers.go
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
			RunAsUser:    ptr.To(int64(1069)),
			RunAsGroup:   ptr.To(int64(1069)),
			RunAsNonRoot: ptr.To(true),
		},
	}
}
