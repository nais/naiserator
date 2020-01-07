package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1alpha3"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GCPSqlInstanceSecretName(instanceName string) string {
	return fmt.Sprintf("sqlinstanceuser-%s", instanceName)
}

func GoogleSqlUser(app *nais.Application, instanceName string, cascadingDelete bool, password string) *google_sql_crd.SQLUser {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = app.Name

	objectMeta.Annotations = CascadingDeleteAnnotation(cascadingDelete)

	return &google_sql_crd.SQLUser{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLUser",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLUserSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instanceName},
			Host:        "%",
			Password:    password,
		},
	}
}
