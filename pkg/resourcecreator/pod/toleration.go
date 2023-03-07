package pod

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TolerationType string

func (t TolerationType) String() string {
	return string(t)
}

const (
	TolerationTypeGKE TolerationType = "cloud.google.com/gke-spot"
)

func nodeAffinity(key, value string) *corev1.NodeAffinity {
	return &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      key,
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{value},
						},
					},
				},
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
							Values:   []string{appName}},
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			}},
		},
	}
}

func ConfigureAffinity(appName string, tolerations []corev1.Toleration) *corev1.Affinity {
	if tolerations == nil {
		return &corev1.Affinity{PodAntiAffinity: appAffinity(appName)}
	}

	a := &corev1.Affinity{}
	for _, toleration := range tolerations {
		switch toleration.Key {
		case TolerationTypeGKE.String():
			a = &corev1.Affinity{
				NodeAffinity:    nodeAffinity(toleration.Key, toleration.Value),
				PodAntiAffinity: appAffinity(appName),
			}
		}
	}
	return a
}

func SetupToleration(toleration config.Toleration) []corev1.Toleration {
	if toleration.EnableSpot {
		return []corev1.Toleration{
			{
				Key:      TolerationTypeGKE.String(),
				Operator: corev1.TolerationOpEqual,
				Value:    "true",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		}
	}
	return nil
}
