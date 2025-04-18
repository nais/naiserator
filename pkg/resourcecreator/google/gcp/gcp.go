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
	google_iam.Config
	google_sql.Config
	google_storagebucket.Config
	google_bigquery.Config
	IsCNRMEnabled() bool
}

type Source interface {
	resource.Source
	GetGCP() *nais_io_v1.GCP
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	gcp := source.GetGCP()

	if !cfg.IsCNRMEnabled() && gcp == nil {
		return nil
	}

	if !cfg.IsCNRMEnabled() && gcp != nil {
		return fmt.Errorf("GCP resources requested, but CNRM is not enabled (not running on GCP?)")
	}

	projectID := cfg.GetGoogleProjectID()
	teamProjectID := cfg.GetGoogleTeamProjectID()

	googleServiceAccount := google_iam.CreateServiceAccount(source, projectID)
	googleServiceAccountBinding := google_iam.CreatePolicy(source, &googleServiceAccount, projectID)

	ast.PrependEnv([]v1.EnvVar{
		// Standard environment variable name in Google SDKs
		{
			Name:  "GOOGLE_CLOUD_PROJECT",
			Value: teamProjectID,
		},
		// Legacy environment variable for backwards compability
		{
			Name:  "GCP_TEAM_PROJECT_ID",
			Value: teamProjectID,
		},
	}...)

	ast.AppendOperation(resource.OperationCreateIfNotExists, &googleServiceAccount)
	ast.AppendOperation(resource.OperationCreateIfNotExists, &googleServiceAccountBinding)

	if !cfg.IsCNRMEnabled() || gcp == nil {
		return nil
	}

	err := google_storagebucket.Create(source, ast, cfg, googleServiceAccount)
	if err != nil {
		return err
	}
	err = google_bigquery.CreateDataset(source, ast, cfg, googleServiceAccount.Name)
	if err != nil {
		return err
	}
	err = google_sql.CreateInstance(source, ast, cfg)
	if err != nil {
		return err
	}
	err = google_iam.CreatePolicyMember(source, ast, cfg)
	if err != nil {
		return err
	}

	return nil
}
