package google_sql

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleSQLDatabase(objectMeta metav1.ObjectMeta, db nais.CloudSqlDatabase, instance nais.CloudSqlInstance, projectId string) *google_sql_crd.SQLDatabase {
	objectMeta.Name = db.Name
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)

	if !instance.CascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
	}

	return &google_sql_crd.SQLDatabase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SqlDatabase",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLDatabaseSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instance.Name},
		},
	}
}
