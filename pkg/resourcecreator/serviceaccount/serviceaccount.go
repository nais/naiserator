package serviceaccount

import (
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(objectMeta metav1.ObjectMeta, options resource.Options, operations *resource.Operations, appNamespaceHash string) {
	if len(options.GoogleProjectId) > 0 {
		objectMeta.Annotations["iam.gke.io/gcp-service-account"] = google.GcpServiceAccountName(appNamespaceHash, options.GoogleProjectId)
	}

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
	}

	*operations = append(*operations, resource.Operation{Resource: serviceAccount, Operation: resource.OperationCreateIfNotExists})
}
