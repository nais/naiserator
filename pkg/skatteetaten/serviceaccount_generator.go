package generator

import (
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateServiceAccount(application v1alpha1.Application) *v1.ServiceAccount {
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
