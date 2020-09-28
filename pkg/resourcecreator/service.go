package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Service(app *nais.Application) *corev1.Service {
	portName := nais.DefaultPortName
	if app.Spec.Service.AppProtocol != "" {
		portName = app.Spec.Service.AppProtocol
	}
	return &corev1.Service{
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
					Name:     portName,
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
}
