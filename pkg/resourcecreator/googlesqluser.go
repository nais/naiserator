package resourcecreator

import (
	"fmt"
	"strings"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	googleSQLHostSuffix     = "_HOST"
	googleSQLPortSuffix     = "_PORT"
	googleSQLUsernameSuffix = "_USERNAME"
	googleSQLPasswordSuffix = "_PASSWORD"
	googleSQLDatabaseSuffix = "_DATABASE"
	googleSQLURLSuffix      = "_URL"

	googleSQLPostgresHost = "127.0.0.1"
	googleSQLPostgresPort = "5432"
	googleSQLPostgresURL  = "postgres://%s:%s@%s:%s/%s"
)

func googleSQLDatabaseCase(x string) string {
	return strings.ReplaceAll(strings.ToUpper(x), "-", "_")
}

func googleSQLPrefix(instance *google_sql_crd.SQLInstance, db *google_sql_crd.SQLDatabase) string {
	instanceName := googleSQLDatabaseCase(instance.Name)
	databaseName := googleSQLDatabaseCase(db.Name)
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", instanceName, databaseName)
}

func GoogleSQLEnvVars(instance *google_sql_crd.SQLInstance, db *google_sql_crd.SQLDatabase, username, password string) map[string]string {
	prefix := googleSQLPrefix(instance, db)
	return map[string]string{
		prefix + googleSQLUsernameSuffix: username,
		prefix + googleSQLPasswordSuffix: password,
		prefix + googleSQLHostSuffix:     googleSQLPostgresHost,
		prefix + googleSQLPortSuffix:     googleSQLPostgresPort,
		prefix + googleSQLDatabaseSuffix: db.Name,
		prefix + googleSQLURLSuffix:      fmt.Sprintf(googleSQLPostgresURL, username, password, googleSQLPostgresHost, googleSQLPostgresPort, db.Name),
	}
}

func GoogleSQLSecretName(app *nais.Application) string {
	return fmt.Sprintf("google-sql-%s", app.Name)
}

func GoogleSqlUser(app *nais.Application, instance *google_sql_crd.SQLInstance, db *google_sql_crd.SQLDatabase, cascadingDelete bool, projectId string) *google_sql_crd.SQLUser {
	objectMeta := app.CreateObjectMeta()

	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)

	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}

	return &google_sql_crd.SQLUser{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLUser",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLUserSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instance.Name},
			Password: google_sql_crd.SqlUserPasswordValue{
				ValueFrom: google_sql_crd.SqlUserPasswordSecretKeyRef{
					SecretKeyRef: google_sql_crd.SecretRef{
						Key:  googleSQLPrefix(instance, db),
						Name: GoogleSQLSecretName(app),
					},
				},
			},
		},
	}
}
