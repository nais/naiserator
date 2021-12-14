package gcp

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_bigquery "github.com/nais/naiserator/pkg/resourcecreator/google/bigquery"
	google_iam "github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	google_storagebucket "github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	v1 "k8s.io/api/core/v1"
)

type Config interface {
	google_sql.Config
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisGCP *nais_io_v1.GCP, cfg Config) error {
	if naisGCP != nil && len(resourceOptions.GoogleTeamProjectId) == 0 {
		// We're not currently in a team namespace with corresponding GCP team project
		return fmt.Errorf("GCP resources requested, but no team project ID annotation set on namespace %s (not running on GCP?)", source.GetNamespace())
	}

	if !resourceOptions.CNRMEnabled && len(resourceOptions.GoogleProjectId) <= 0 {
		return nil
	}

	googleServiceAccount := google_iam.CreateServiceAccount(source, resourceOptions.GoogleProjectId)
	googleServiceAccountBinding := google_iam.CreatePolicy(source, &googleServiceAccount, resourceOptions.GoogleProjectId)
	ast.Env = append(ast.Env, v1.EnvVar{
		Name:  "GCP_TEAM_PROJECT_ID",
		Value: resourceOptions.GoogleTeamProjectId,
	})

	ast.AppendOperation(resource.OperationCreateIfNotExists, &googleServiceAccount)
	ast.AppendOperation(resource.OperationCreateIfNotExists, &googleServiceAccountBinding)

	if resourceOptions.CNRMEnabled && naisGCP != nil {
		err := google_storagebucket.Create(source, ast, resourceOptions, googleServiceAccount, naisGCP.Buckets)
		if err != nil {
			return err
		}
		err = google_bigquery.CreateDataset(source, ast, resourceOptions, naisGCP.BigQueryDatasets, googleServiceAccount.Name)
		if err != nil {
			return err
		}
		err = google_sql.CreateInstance(source, ast, resourceOptions, &naisGCP.SqlInstances, cfg)
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
