package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1alpha3"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleSqlDatabases(app *nais.Application, instance nais.CloudSqlInstance) (databases []*google_sql_crd.SQLDatabase) {
	for _, db := range instance.Databases {
		databases = append(databases, googleSqlDatabase(db.Name, instance.Name, instance.CascadingDelete, app))
	}
	return
}

func googleSqlDatabase(name, instanceName string, cascadingDelete bool, app *nais.Application) *google_sql_crd.SQLDatabase {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = name

	objectMeta.Annotations = CascadingDeleteAnnotation(cascadingDelete)

	return &google_sql_crd.SQLDatabase{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SqlDatabase",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLDatabaseSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instanceName},
		},
	}
}
