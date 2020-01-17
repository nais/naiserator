package resourcecreator

import (
	"fmt"
	"strings"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GCPSqlInstanceSecretName(instanceName string) string {
	return fmt.Sprintf("sqlinstanceuser-%s", instanceName)
}

func GoogleSqlUser(app *nais.Application, instanceName string, cascadingDelete bool, password string) *google_sql_crd.SQLUser {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = app.Name

	if !cascadingDelete {
		ApplyAbandonDeletionPolicy(&objectMeta)
	}

	casedInstanceName := strings.ReplaceAll(strings.ToUpper(instanceName), "-", "_")

	return &google_sql_crd.SQLUser{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLUser",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLUserSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instanceName},
			Host:        "%",
			Password: google_sql_crd.SqlUserPasswordValue{
				ValueFrom: google_sql_crd.SqlUserPasswordSecretKeyRef{
					SecretKeyRef: google_sql_crd.SecretRef{
						Key:  fmt.Sprintf("GCP_SQLINSTANCE_%s_PASSWORD", casedInstanceName),
						Name: GCPSqlInstanceSecretName(instanceName),
					},
				},
			},
		},
	}
}

func GoogleSqlUserEnvVars(instanceName, password string) map[string]string {
	cased := strings.ReplaceAll(strings.ToUpper(instanceName), "-", "_")
	return map[string]string{
		fmt.Sprintf("GCP_SQLINSTANCE_%s_PASSWORD", cased): password,
		fmt.Sprintf("GCP_SQLINSTANCE_%s_USERNAME", cased): instanceName,
	}
}
