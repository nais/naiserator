package resourcecreator

import (
	"github.com/golang/glog"
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net/url"
)

func ingress(app *nais.Application) *extensionsv1beta1.Ingress {
	var rules []extensionsv1beta1.IngressRule

	for _, ingress := range app.Spec.Ingresses {
		parsedUrl, err := url.Parse(ingress)
		if err != nil {
			glog.Errorf("Failed to parse url: %s. Error was: %s", ingress, err)
		}

		rules = append(rules, extensionsv1beta1.IngressRule{
			Host: parsedUrl.Host,
			IngressRuleValue: extensionsv1beta1.IngressRuleValue{
				HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
					Paths: []extensionsv1beta1.HTTPIngressPath{
						{
							Path: parsedUrl.Path,
							Backend: extensionsv1beta1.IngressBackend{
								ServiceName: app.Name,
								ServicePort: intstr.IntOrString{IntVal: nais.DefaultPort},
							},
						},
					},
				},
			},
		})
	}

	return &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: extensionsv1beta1.IngressSpec{
			Rules: rules,
		},
	}
}
