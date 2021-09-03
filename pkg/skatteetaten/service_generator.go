package generator

import (
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateService(application v1alpha1.Application) *corev1.Service {

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
