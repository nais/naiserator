package generator

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateServiceAccount(application skatteetaten_no_v1alpha1.Application) *v1.ServiceAccount {
	serviceAccount := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: application.StandardObjectMeta(),
		ImagePullSecrets: []v1.LocalObjectReference{{
			Name: "ghcr-secret",
		}},
	}
	return serviceAccount
}
