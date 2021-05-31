package gcp

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisGCP *nais_io_v1alpha1.GCP) error {
	if len(resourceOptions.GoogleProjectId) <= 0 {
		return nil
	}

	googleServiceAccount := google_iam.CreateServiceAccount(source, resourceOptions.GoogleProjectId)
	googleServiceAccountBinding := google_iam.CreatePolicy(source, &googleServiceAccount, resourceOptions.GoogleProjectId)
	ast.Operations = append(ast.Operations, resource.Operation{Resource: &googleServiceAccount, Operation: resource.OperationCreateOrUpdate})
	ast.Operations = append(ast.Operations, resource.Operation{Resource: &googleServiceAccountBinding, Operation: resource.OperationCreateOrUpdate})

	if naisGCP != nil {
		google_storagebucket.Create(source, ast, resourceOptions, googleServiceAccount, naisGCP.Buckets)
		err := google_sql.CreateInstance(source, ast, resourceOptions, &naisGCP.SqlInstances)
		if err != nil {
			return err
		}
		err = google_iam.CreatePolicyMember(source, ast, resourceOptions, naisGCP.Permissions)
		if err != nil {
			return err
		}
	}

	return nil
}
