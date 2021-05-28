package aiven

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

func Elastic(deployment *appsv1.Deployment, elastic *nais_io_v1alpha1.Elastic) {
	if elastic == nil {
		return
	}

	deployment.Spec.Template.ObjectMeta.Labels["aiven"] = "enabled"
}
