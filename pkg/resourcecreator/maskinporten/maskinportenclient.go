package maskinporten

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func secretName(appName string) string {
	return namegen.PrefixedRandShortName("maskinporten", appName, validation.DNS1035LabelMaxLength)
}

func client(objectMeta metav1.ObjectMeta, naisMaskinporten *nais_io_v1.Maskinporten) *nais_io_v1.MaskinportenClient {
	return &nais_io_v1.MaskinportenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MaskinportenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.MaskinportenClientSpec{
			Scopes: nais_io_v1.MaskinportenScope{
				ConsumedScopes: naisMaskinporten.Scopes.ConsumedScopes,
				ExposedScopes:  naisMaskinporten.Scopes.ExposedScopes,
			},
			SecretName: secretName(objectMeta.Name),
		},
	}
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisMaskinporten *nais_io_v1.Maskinporten) {
	if !resourceOptions.DigdiratorEnabled || naisMaskinporten == nil || !naisMaskinporten.Enabled {
		return
	}

	maskinportenClient := client(resource.CreateObjectMeta(source), naisMaskinporten)

	ast.AppendOperation(resource.OperationCreateOrUpdate, maskinportenClient)
	pod.WithAdditionalSecret(ast, maskinportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, maskinportenClient.Spec.SecretName)
}
