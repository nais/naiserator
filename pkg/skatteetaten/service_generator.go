package generator

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateService(application skatteetaten_no_v1alpha1.Application) *corev1.Service {

	//finn service i clusteret
	//hvis den har satt nap-skip-reconcyle, ikke gjør noe.
	//hvis ikke diff og se om man må gjøre noe

	//TODO: prometheus
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     8080,
				},
			},
			Selector: application.StandardLabelSelector(),
		},
	}

	return service
}
