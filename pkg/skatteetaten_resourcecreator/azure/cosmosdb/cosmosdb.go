package cosmosdb

import (
	"fmt"
	"strings"

	azure_microsoft_com_v1alpha1 "github.com/nais/liberator/pkg/apis/azure.microsoft.com/v1alpha1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Source interface {
	resource.Source
	GetAzureResourceGroup() string
	GetCosmosDb() map[string]*skatteetaten_no_v1alpha1.CosmosDBConfig
}

func Create(app Source, ast *resource.Ast) {
	cosmosDb := app.GetCosmosDb()
	resourceGroup := app.GetAzureResourceGroup()
	single := false
	if len(cosmosDb) == 1 {
		single = true
	}

	for _, db := range cosmosDb {
		generateCosmosDb(app, ast, resourceGroup, db, single)
	}
}

func generateCosmosDb(source resource.Source, ast *resource.Ast, rg string, db *skatteetaten_no_v1alpha1.CosmosDBConfig, single bool) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("cod-%s-%s-%s", source.GetNamespace(), source.GetName(), db.Name)

	spec := azure_microsoft_com_v1alpha1.CosmosDBSpec{
		Location:      "norwayeast",
		ResourceGroup: rg,
		Properties: azure_microsoft_com_v1alpha1.CosmosDBProperties{
			DatabaseAccountOfferType: "Standard",
		},
	}
	if db.MongoDBVersion != "" {
		spec.Kind = "MongoDB"
		spec.Properties.MongoDBVersion = db.MongoDBVersion
		spec.Properties.Capabilities = &[]azure_microsoft_com_v1alpha1.Capability{{
			Name: pointer.StringPtr("EnableMongo"),
		}}
	} else {
		spec.Kind = "GlobalDocumentDB"
	}

	object := &azure_microsoft_com_v1alpha1.CosmosDB{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CosmosDB",
			APIVersion: "azure.microsoft.com/v1alpha1",
		},
		ObjectMeta: objectMeta,
		Spec:       spec,
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, object)
	envVar := createConnectionStringEnvVar(objectMeta, db, single)
	ast.Env = append(ast.Env, envVar...)
}

func createConnectionStringEnvVar(objectMeta metav1.ObjectMeta, db *skatteetaten_no_v1alpha1.CosmosDBConfig, single bool) []corev1.EnvVar {
	secretName := fmt.Sprintf("cosmosdb-%s", objectMeta.Name)

	prefix := "SPRING_DATA_MONGODB"
	if db.MongoDBVersion == "" {
		prefix = "COSMOSDB"
	}

	if !single {
		envVar := corev1.EnvVar{
			Name: fmt.Sprintf("%s_%s_URI", prefix, strings.ToUpper(db.Name)),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "connectionString0",
				},
			},
		}
		return []corev1.EnvVar{envVar}
	}

	uri := corev1.EnvVar{
		Name: fmt.Sprintf("%s_URI", prefix),
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: "connectionString0",
			},
		},
	}

	name := corev1.EnvVar{
		Name:  fmt.Sprintf("%s_DATABASE", prefix),
		Value: db.Name,
	}
	return []corev1.EnvVar{uri, name}

}
