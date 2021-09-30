package skatteetaten_generator


import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Source interface {
	resource.Source
	GetReplicas() *nais_io_v1.Replicas
	GetImagePolicy() *skatteetaten_no_v1alpha1.ImagePolicyConfig
	GetIngress() *skatteetaten_no_v1alpha1.IngressConfig
	GetEgress() *skatteetaten_no_v1alpha1.EgressConfig

	GetAzureResourceGroup() string
	GetPostgresDatabases() []*skatteetaten_no_v1alpha1.PostgreDatabaseConfig
}

