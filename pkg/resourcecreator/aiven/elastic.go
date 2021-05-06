package aiven

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

func Elastic(app *nais_io_v1alpha1.Application, deployment *appsv1.Deployment) {
	if app.Spec.Elastic == nil {
		return
	}

	deployment.Spec.Template.ObjectMeta.Labels["aiven"] = "enabled"
}
