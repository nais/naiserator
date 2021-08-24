package service

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1_alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisService nais_io_v1.Service) {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": source.GetName()},
			Ports: []corev1.ServicePort{
				{
					Name:     naisService.Protocol,
					Protocol: corev1.ProtocolTCP,
					Port:     naisService.Port,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: nais_io_v1_alpha1.DefaultPortName,
					},
				},
			},
		},
	}

	if resourceOptions.WonderwallEnabled {
		service.Spec.Ports[0].TargetPort = intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 8090,
		}
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, service)
}
