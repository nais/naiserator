package serviceaccount

import (
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(source resource.Source, ast *resource.Ast, options resource.Options) {
	objectMeta := source.CreateObjectMeta()
	if len(options.GoogleProjectId) > 0 {
		objectMeta.Annotations["iam.gke.io/gcp-service-account"] = google.GcpServiceAccountName(source.CreateAppNamespaceHash(), options.GoogleProjectId)
	}

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, serviceAccount)
}
