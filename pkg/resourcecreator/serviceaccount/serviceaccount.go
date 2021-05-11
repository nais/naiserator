package serviceaccount

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(app *nais.Application, options resource.Options, operations *resource.Operations) {
	objectMeta := app.CreateObjectMeta()
	if len(options.GoogleProjectId) > 0 {
		objectMeta.Annotations["iam.gke.io/gcp-service-account"] = google.GcpServiceAccountName(app, options.GoogleProjectId)
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
