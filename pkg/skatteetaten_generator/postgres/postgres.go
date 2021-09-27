package postgres

import (
	"fmt"

	azure_microsoft_com_v1alpha1 "github.com/nais/liberator/pkg/apis/azure.microsoft.com/v1alpha1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/postgres_env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: where do we want name generation?
func Create(source resource.Source, ast *resource.Ast, pgd []*skatteetaten_no_v1alpha1.PostgreDatabaseConfig, resourceGroup string) {

	// TODO handle updating
	for dbIndex, db := range pgd {
		generatePostgresDatabase(source, ast, resourceGroup, *db)
		for userIndex, user := range db.Users {
			if dbIndex == 0 && userIndex == 0 {
				secretName := fmt.Sprintf("postgresqluser-pgu-%s-%s", source.GetName(), user.Name)
				dbVars := postgres_env.GenerateDbEnv("SPRING_DATASOURCE", secretName)
				ast.Env = append(ast.Env, dbVars...)
			}
			generatePostgresUser(source, ast, resourceGroup, *db, *user)
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
