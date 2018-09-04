package resourcecreator

import (
	"github.com/nais/naiserator/api/types/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateResourceSpecs(app *v1alpha1.Application) ([]interface{}, error) {
	var resources []interface{}
	resources = append(resources, createServiceSpec(app))
	return resources, nil
}

func createServiceSpec(app *v1alpha1.Application) *corev1.Service {
	blockOwnerDeletion := true
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "v1alpha1",
				Kind:               "Application",
				Name:               app.Name,
				UID:                app.UID,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}}},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 69,
				},
			},
		},
	}
}
