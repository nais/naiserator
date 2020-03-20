package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleSqlDatabases(app *nais.Application, instance nais.CloudSqlInstance, projectId string) (databases []*google_sql_crd.SQLDatabase) {
	for _, db := range instance.Databases {
		databases = append(databases, googleSqlDatabase(db.Name, instance.Name, instance.CascadingDelete, app, projectId))
	}
	return
}

func googleSqlDatabase(name, instanceName string, cascadingDelete bool, app *nais.Application, projectId string) *google_sql_crd.SQLDatabase {
	objectMeta := app.CreateObjectMetaWithName(name)

	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)

	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}

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
