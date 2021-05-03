package google_sql

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleSQLDatabase(app *nais.Application, db nais.CloudSqlDatabase, instance nais.CloudSqlInstance, projectId string) *google_sql_crd.SQLDatabase {
	objectMeta := app.CreateObjectMetaWithName(db.Name)

	util.SetAnnotation(&objectMeta, google.GoogleProjectIdAnnotation, projectId)

	if !instance.CascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.GoogleDeletionPolicyAnnotation, google.GoogleDeletionPolicyAbandon)
	}

	return &google_sql_crd.SQLDatabase{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SqlDatabase",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLDatabaseSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instance.Name},
		},
	}
}
