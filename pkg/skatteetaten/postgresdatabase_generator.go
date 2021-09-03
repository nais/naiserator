package generator

import (
	azurev1alpha1 "github.com/Azure/azure-service-operator/api/v1alpha1"
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GeneratePostgresDatabase(application v1alpha1.Application, rg string, database v1alpha1.PostgreDatabaseConfig) *azurev1alpha1.PostgreSQLDatabase {

	db := &azurev1alpha1.PostgreSQLDatabase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQLDatabase",
			APIVersion: "azure.microsoft.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.AzureName(application),
			Namespace: application.Namespace,
			Labels:    application.StandardLabels(),
		},
		Spec: azurev1alpha1.PostgreSQLDatabaseSpec{
			ResourceGroup: rg,
			Server:        database.AzureServerName(application),
		},
	}
	return db

}
