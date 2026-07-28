package service

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1_alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Source interface {
	resource.Source
	GetService() *nais_io_v1.Service
}

func Create(source Source, ast *resource.Ast) {
	svc := source.GetService()

	targetPort := intstr.FromString(nais_io_v1_alpha1.DefaultPortName)
	if ast.ServiceTargetPort != nil {
		targetPort = intstr.FromInt32(*ast.ServiceTargetPort)
	}

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
					Name:       svc.Protocol,
					Protocol:   corev1.ProtocolTCP,
					Port:       svc.Port,
					TargetPort: targetPort,
				},
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, service)
}
