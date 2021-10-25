package postgres

import (
	"fmt"

	azure_microsoft_com_v1alpha1 "github.com/nais/liberator/pkg/apis/azure.microsoft.com/v1alpha1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


type Source interface {
	resource.Source
	GetAzureResourceGroup() string
	GetPostgresDatabases() map[string]*skatteetaten_no_v1alpha1.PostgreDatabaseConfig
}

func Create(app Source, ast *resource.Ast) {

	pgd := app.GetPostgresDatabases()
	resourceGroup := app.GetAzureResourceGroup()
	springDataSourceCreated := false
	for _, db := range pgd {
		generatePostgresDatabase(app, ast, resourceGroup, *db)
		for _, user := range db.Users {
			if !springDataSourceCreated {
				secretName := fmt.Sprintf("postgresqluser-pgu-%s-%s", app.GetName(), user.Name)
				dbVars := GenerateDbEnv("SPRING_DATASOURCE", secretName)
				ast.Env = append(ast.Env, dbVars...)
				springDataSourceCreated=true
			}
			generatePostgresUser(app, ast, resourceGroup, *db, *user)
		}
	}
}

func generatePostgresDatabase(source resource.Source, ast *resource.Ast, rg string, database skatteetaten_no_v1alpha1.PostgreDatabaseConfig) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pgd-%s-%s-%s", source.GetNamespace(), source.GetName(), database.Name)

	db := &azure_microsoft_com_v1alpha1.PostgreSQLDatabase{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQLDatabase",
			APIVersion: "azure.microsoft.com/v1alpha1",
		},
		ObjectMeta: objectMeta,
		Spec: azure_microsoft_com_v1alpha1.PostgreSQLDatabaseSpec{
			ResourceGroup: rg,
			Server:        fmt.Sprintf("pgs-%s-%s", source.GetNamespace(), database.Server),
		},
	}
	ast.AppendOperation(resource.OperationCreateIfNotExists, db)

}

func generatePostgresUser(source resource.Source, ast *resource.Ast, rg string, database skatteetaten_no_v1alpha1.PostgreDatabaseConfig, user skatteetaten_no_v1alpha1.PostgreDatabaseUser) {

	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pgu-%s-%s", source.GetName(), user.Name)

	pgu := &azure_microsoft_com_v1alpha1.PostgreSQLUser{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQLUser",
			APIVersion: "azure.microsoft.com/v1alpha1",
		},
		ObjectMeta: objectMeta,
		Spec: azure_microsoft_com_v1alpha1.PostgreSQLUserSpec{
			DbName:        fmt.Sprintf("pgd-%s-%s-%s", source.GetNamespace(), source.GetName(), database.Name),
			ResourceGroup: rg,
			Server:        fmt.Sprintf("pgs-%s-%s", source.GetNamespace(), database.Server),
			Roles:         []string{user.Role},
		},
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, pgu)
}
