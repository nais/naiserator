package pod

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NaisGarTolerationKey   = "nais.io/gar"
	NaisGarNodeSelectorKey = "nais.io/gar-node-pool"
	GarImagePrefix         = "europe-north1-docker.pkg.dev/"
)

func SetupTolerations(cfg Config, image string) []corev1.Toleration {
	var tolerations []corev1.Toleration

	if cfg.IsGARTolerationEnabled() && strings.HasPrefix(image, GarImagePrefix) {
		tolerations = append(tolerations, corev1.Toleration{
			Key:      NaisGarTolerationKey,
			Operator: corev1.TolerationOpEqual,
			Value:    "true",
			Effect:   corev1.TaintEffectNoSchedule,
		})
	}
	return tolerations
}

func ConfigureAffinity(appName string, tolerations []corev1.Toleration) *corev1.Affinity {
	if tolerations == nil {
		return &corev1.Affinity{PodAntiAffinity: appAffinity(appName)}
	}

	var nodeSelectorTerms []corev1.NodeSelectorTerm

	for _, toleration := range tolerations {
		switch toleration.Key {
		case NaisGarTolerationKey:
			nodeSelectorTerms = append(nodeSelectorTerms, nodeSelectorTerm(NaisGarNodeSelectorKey, toleration.Value))
		}
	}
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
		},
		PodAntiAffinity: appAffinity(appName),
	}
}

func nodeSelectorTerm(key, value string) corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      key,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{value},
			},
		},
	}
}

func appAffinity(appName string) *corev1.PodAntiAffinity {
	return &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{Weight: 10, PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{appName},
						},
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			}},
		},
	}
}
