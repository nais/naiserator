package gcp

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
)

func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) error {
	if len(resourceOptions.GoogleProjectId) <= 0 {
		return nil
	}

	googleServiceAccount := google_iam.ServiceAccount(app, resourceOptions.GoogleProjectId)
	googleServiceAccountBinding := google_iam.Policy(app, &googleServiceAccount, resourceOptions.GoogleProjectId)
	*operations = append(*operations, resource.Operation{Resource: &googleServiceAccount, Operation: resource.OperationCreateOrUpdate})
	*operations = append(*operations, resource.Operation{Resource: &googleServiceAccountBinding, Operation: resource.OperationCreateOrUpdate})

	if app.Spec.GCP != nil {
		google_storagebucket.Create(app, resourceOptions, operations, googleServiceAccount)
		err := google_sql.CreateSqlInstance(app, resourceOptions, deployment, operations)
		if err != nil {
			return err
		}
		err = google_iam.Create(app, resourceOptions, operations)
		if err != nil {
			return err
		}
	}

	return nil
}
