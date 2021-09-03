package generator

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"

	azurev1alpha1 "github.com/Azure/azure-service-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GeneratePostgresUser(application skatteetaten_no_v1alpha1.Application, rg string, database skatteetaten_no_v1alpha1.PostgreDatabaseConfig, user skatteetaten_no_v1alpha1.PostgreDatabaseUser) *azurev1alpha1.PostgreSQLUser {

	return &azurev1alpha1.PostgreSQLUser{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQLUser",
			APIVersion: "azure.microsoft.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.AzureName(application),
			Namespace: application.Namespace,
			Labels:    application.StandardLabels(),
		},
		Spec: azurev1alpha1.PostgreSQLUserSpec{
			DbName:        database.AzureName(application),
			ResourceGroup: rg,
			Server:        database.AzureServerName(application),
			Roles:         []string{user.Role},
		},
	}

}
