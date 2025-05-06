package gcp

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_bigquery "github.com/nais/naiserator/pkg/resourcecreator/google/bigquery"
	google_iam "github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	google_storagebucket "github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
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

	googleServiceAccount := google_iam.CreateServiceAccount(source, projectID)
	googleServiceAccountBinding := google_iam.CreatePolicy(source, &googleServiceAccount, projectID)

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
