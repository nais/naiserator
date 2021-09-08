package generator

import (
	azure_microsoft_com_v1alpha1 "github.com/nais/liberator/pkg/apis/azure.microsoft.com/v1alpha1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GeneratePostgresDatabase(application skatteetaten_no_v1alpha1.Application, rg string, database skatteetaten_no_v1alpha1.PostgreDatabaseConfig) *azure_microsoft_com_v1alpha1.PostgreSQLDatabase {

	db := &azure_microsoft_com_v1alpha1.PostgreSQLDatabase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQLDatabase",
			APIVersion: "azure.microsoft.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.AzureName(application),
			Namespace: application.Namespace,
			Labels:    application.StandardLabels(),
		},
		Spec: azure_microsoft_com_v1alpha1.PostgreSQLDatabaseSpec{
			ResourceGroup: rg,
			Server:        database.AzureServerName(application),
		},
	}
	return db

}
