package postgres

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Max length is 63, but we need to save space for suffixes added by Zalando operator or StatefulSets
	maxClusterNameLength = 50
)

type Source interface {
	resource.Source
	GetPostgres() *nais_io_v1.Postgres
}

type Config interface {
	GetGoogleProjectID() string
	PostgresStorageClass() string
	PostgresImage() string
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	postgres := source.GetPostgres()
	if postgres == nil {
		return nil
	}

	var err error
	pgClusterName := postgres.ClusterName
	if len(pgClusterName) > maxClusterNameLength {
		pgClusterName, err = namegen.ShortName(pgClusterName, maxClusterNameLength)
		if err != nil {
			return fmt.Errorf("failed to shorten PostgreSQL cluster name: %w", err)
		}
	}
	pgNamespace := fmt.Sprintf("pg-%s", source.GetNamespace())

	createNetworkPolicies(source, ast, pgClusterName, pgNamespace)

	envVars := []corev1.EnvVar{
		{
			Name:  "PGHOST",
			Value: fmt.Sprintf("%s-pooler.%s", pgClusterName, pgNamespace),
		},
		{
			Name:  "PGPORT",
			Value: "5432",
		},
		{
			Name:  "PGDATABASE",
			Value: "app",
		},
		{
			Name: "PGUSER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "username",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName),
					},
				},
			},
		},
		{
			Name: "PGPASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "password",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName),
					},
				},
			},
		},
		{
			Name:  "PGURL",
			Value: fmt.Sprintf("postgresql://$(PGUSER):$(PGPASSWORD)@%s-pooler.%s:5432/app", pgClusterName, pgNamespace),
		},
		{
			Name:  "PGJDBCURL",
			Value: fmt.Sprintf("jdbc:postgresql://%s-pooler.%s:5432/app?user=$(PGUSER)&password=$(PGPASSWORD)", pgClusterName, pgNamespace),
		},
	}

	ast.Env = append(ast.Env, envVars...)

	return nil
}
