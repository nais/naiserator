package zalando

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"k8s.io/api/core/v1"
)

func Create(source resource.Source, ast *resource.Ast, postgres *nais_io_v1.Postgres) {
	pgClusterName := postgres.ClusterName
	pgNamespace := fmt.Sprintf("pg-%s", source.GetNamespace())

	createNetworkPolicies(source, ast, pgClusterName, pgNamespace)

	envVars := []v1.EnvVar{
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
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					Key: "username",
					LocalObjectReference: v1.LocalObjectReference{
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName),
					},
				},
			},
		},
		{
			Name: "PGPASSWORD",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					Key: "password",
					LocalObjectReference: v1.LocalObjectReference{
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
}
