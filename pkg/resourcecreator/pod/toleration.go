package pod

import (
	"fmt"

	"github.com/nais/naiserator/pkg/naiserator/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TolerationType string

const (
	TolerationTypeNais = "cloud.google.com/gke-spot"
	TolerationTypeGKE  = "nais.io/node-type"
)

func mapTolerations(tolerations []config.Toleration) []corev1.Toleration {
	var ts []corev1.Toleration

	for _, toleration := range tolerations {
		ts = append(ts, corev1.Toleration{
			Key:      toleration.Key,
			Operator: toleration.Operator,
			Value:    toleration.Value,
			Effect:   toleration.Effect,
			TolerationSeconds: func() *int64 {
				if toleration.TolerationSeconds != nil {
					return toleration.TolerationSeconds
				}
				return nil
			}(),
		})
	}
	return ts
}

func nodeAffinity(toleration corev1.Toleration) *corev1.NodeAffinity {
	return &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      toleration.Key,
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{toleration.Value},
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
	if len(tolerations) == 0 {
		return &corev1.Affinity{PodAntiAffinity: appAffinity(appName)}
	}

	a := &corev1.Affinity{}
	for _, toleration := range tolerations {
		if toleration.Key == TolerationTypeGKE || toleration.Key == TolerationTypeNais {
			a = &corev1.Affinity{
				NodeAffinity:    nodeAffinity(toleration),
				PodAntiAffinity: appAffinity(appName),
			}
		}
	}
	return a
}

func contains(tolerations []config.Toleration, key string) bool {
	for _, toleration := range tolerations {
		if toleration.Key == key {
			return true
		}
	}
	return false
}

func SetupTolerations(tolerations []config.Toleration) ([]corev1.Toleration, error) {

	if len(tolerations) > 1 && contains(tolerations, TolerationTypeGKE) && contains(tolerations, TolerationTypeNais) {
		return nil, fmt.Errorf("cannot have both %s and %s tolerations", TolerationTypeGKE, TolerationTypeNais)
	}

	return mapTolerations(tolerations), nil
}
