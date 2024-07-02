package google_sql

import (
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func CreateGoogleSQLDatabase(objectMeta metav1.ObjectMeta, instanceName, dbName, projectId string, cascadingDelete bool) *google_sql_crd.SQLDatabase {
	// Spec for CloudSqlDatabase states that Name is required
	var err error

	objectMeta.Name, err = namegen.ShortName(objectMeta.GetName()+"-"+dbName, validation.DNS1035LabelMaxLength)
	if err != nil {
		panic(err) // never happens
	}
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)

	// This is an annotation, but also a spec field.
	// Which one has presedence? We set both to be certain.
	// var deletionPolicy string
	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
		// deletionPolicy = google_sql_crd.DeletionPolicyAbandon
	} else {
		// deletionPolicy = google_sql_crd.DeletionPolicyDelete
	}

	return &google_sql_crd.SQLDatabase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SQLDatabase",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLDatabaseSpec{
			InstanceRef: google_sql_crd.InstanceRef{Name: instanceName},
			ResourceID:  dbName,
			// DeletionPolicy is part of the spec, but setting it to anything else than DELETE seems to break with:
			// {"severity":"error","logger":"sqldatabase-controller","msg":"error applying desired state","error":"summary: \n"}
			// DeletionPolicy: deletionPolicy,
		},
	}
}
