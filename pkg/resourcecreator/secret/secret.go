package secret

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func OpaqueSecret(objectMeta metav1.ObjectMeta, secretName string, secrets map[string]string) *corev1.Secret {
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
