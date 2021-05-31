package maskinporten

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func client(app *nais_io_v1alpha1.Application) *nais_io_v1.MaskinportenClient {
	return &nais_io_v1.MaskinportenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MaskinportenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: nais_io_v1.MaskinportenClientSpec{
			Scopes: nais_io_v1.MaskinportenScope{
				ConsumedScopes: app.Spec.Maskinporten.Scopes.ConsumedScopes,
				ExposedScopes:  app.Spec.Maskinporten.Scopes.ExposedScopes,
			},
			SecretName: maskinPortenSecretName(*app),
		},
	}
}

func maskinPortenSecretName(app nais_io_v1alpha1.Application) string {
	return namegen.PrefixedRandShortName("maskinporten", app.Name, validation.DNS1035LabelMaxLength)
}

func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) {
	if resourceOptions.DigdiratorEnabled && app.Spec.Maskinporten != nil && app.Spec.Maskinporten.Enabled {
		maskinportenClient := client(app)

		*operations = append(*operations, resource.Operation{Resource: maskinportenClient, Operation: resource.OperationCreateOrUpdate})

		podSpec := &deployment.Spec.Template.Spec
		podSpec = pod.WithAdditionalSecret(podSpec, maskinportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, maskinportenClient.Spec.SecretName)
		deployment.Spec.Template.Spec = *podSpec
	}
	return
}
