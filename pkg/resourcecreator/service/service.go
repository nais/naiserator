package service

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Create(app *nais.Application, operations *resource.Operations) {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": app.Name},
			Ports: []corev1.ServicePort{
				{
					Name:     app.Spec.Service.Protocol,
					Protocol: corev1.ProtocolTCP,
					Port:     app.Spec.Service.Port,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: nais.DefaultPortName,
					},
				},
			},
		},
	}

	*operations = append(*operations, resource.Operation{Resource: service, Operation: resource.OperationCreateOrUpdate})
}
