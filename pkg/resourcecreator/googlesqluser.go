package resourcecreator

import (
	"fmt"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
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

func googleSQLPrefix(db *nais.CloudSqlDatabase, instanceName string) string {
	if len(db.EnvVarPrefix) > 0 {
		return strings.TrimSuffix(db.EnvVarPrefix, "_")
	}
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", googleSQLDatabaseCase(instanceName), googleSQLDatabaseCase(db.Name))
}

func firstKeyWithSuffix(vars map[string]string, suffix string) (string, error) {
	for k := range vars {
		if strings.HasSuffix(k, suffix) {
			return k, nil
		}
	}
	return "", fmt.Errorf("no variable found matching suffix %s", suffix)
}

func mergeStandardUserWithAdditional(additionalUsers []nais.AdditionalUser, sqlUserName string) []nais.AdditionalUser {
	standardUser := nais.AdditionalUser{
		Name: sqlUserName,
	}

	if additionalUsers != nil {
		return append(additionalUsers, standardUser)
	} else {
		return []nais.AdditionalUser{
			standardUser,
		}
	}
}

func GoogleSQLEnvVars(db *nais.CloudSqlDatabase, instanceName, username, password string) map[string]string {
	var prefix string

	prefix = googleSQLPrefix(db, instanceName)

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

func GoogleSqlUser(app *nais.Application, instance *google_sql_crd.SQLInstance, secretKeyRefEnvName string, cascadingDelete bool, projectId string, sqlUserName string) *google_sql_crd.SQLUser {
	objectMetadata := app.CreateObjectMetaWithName(sqlUserName)
	setAnnotations(objectMetadata, cascadingDelete, projectId)
	return createSQLUser(app, objectMetadata, instance.Name, secretKeyRefEnvName)
}

func setAnnotations(objectMeta k8s_meta.ObjectMeta, cascadingDelete bool, projectId string) {
	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)
	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}
}

func createSQLUser(app *nais.Application, objectMeta k8s_meta.ObjectMeta, instanceName string, secretKeyRefEnvName string) *google_sql_crd.SQLUser {
	return &google_sql_crd.SQLUser{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLUser",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLUserSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instanceName},
			Password: google_sql_crd.SqlUserPasswordValue{
				ValueFrom: google_sql_crd.SqlUserPasswordSecretKeyRef{
					SecretKeyRef: google_sql_crd.SecretRef{
						Key:  secretKeyRefEnvName,
						Name: GoogleSQLSecretName(app),
					},
				},
			},
		},
	}
}
