package serviceaccount

import (
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config interface {
	GetGoogleProjectID() string
	IsGCPEnabled() bool
}

func Create(source resource.Source, ast *resource.Ast, cfg Config) {
	objectMeta := resource.CreateObjectMeta(source)

	if cfg.IsGCPEnabled() {
		googleProjectID := cfg.GetGoogleProjectID()
		objectMeta.Annotations["iam.gke.io/gcp-service-account"] = google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), googleProjectID)
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
