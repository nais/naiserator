package gcp

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations, appNamespaceHash string, naisGCP *nais_io_v1alpha1.GCP) error {
	if len(resourceOptions.GoogleProjectId) <= 0 {
		return nil
	}

	googleServiceAccount := google_iam.CreateServiceAccount(objectMeta, resourceOptions.GoogleProjectId, appNamespaceHash)
	googleServiceAccountBinding := google_iam.CreatePolicy(objectMeta, &googleServiceAccount, resourceOptions.GoogleProjectId, appNamespaceHash)
	*operations = append(*operations, resource.Operation{Resource: &googleServiceAccount, Operation: resource.OperationCreateOrUpdate})
	*operations = append(*operations, resource.Operation{Resource: &googleServiceAccountBinding, Operation: resource.OperationCreateOrUpdate})

	if naisGCP != nil {
		google_storagebucket.Create(objectMeta, resourceOptions, operations, googleServiceAccount, appNamespaceHash, naisGCP.Buckets)
		err := google_sql.CreateInstance(objectMeta, resourceOptions, deployment, operations, &naisGCP.SqlInstances, appNamespaceHash)
		if err != nil {
			return err
		}
		err = google_iam.CreatePolicyMember(objectMeta, resourceOptions, operations, appNamespaceHash, naisGCP.Permissions)
		if err != nil {
			return err
		}
	}

	return nil
}
