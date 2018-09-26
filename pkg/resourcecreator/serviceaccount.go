package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func serviceAccount(app *nais.Application) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
	}
}
