package resourcecreator

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func OpaqueSecret(app *nais.Application, secretName string, secrets map[string]string) *corev1.Secret {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = secretName
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
		StringData: secrets,
		Type:       "Opaque",
	}
}
