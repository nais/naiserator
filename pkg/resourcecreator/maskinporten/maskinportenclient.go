package maskinporten

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	v12 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func client(app *nais.Application) *v1.MaskinportenClient {
	return &v1.MaskinportenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MaskinportenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: v1.MaskinportenClientSpec{
			Scopes:     app.Spec.Maskinporten.Scopes,
			SecretName: maskinPortenSecretName(*app),
		},
	}
}

func maskinPortenSecretName(app nais.Application) string {
	return namegen.PrefixedRandShortName("maskinporten", app.Name, validation.DNS1035LabelMaxLength)
}

func Create(app *nais.Application, resourceOptions resource.Options, deployment *v12.Deployment, operations *resource.Operations) {
	if resourceOptions.DigdiratorEnabled && app.Spec.Maskinporten != nil && app.Spec.Maskinporten.Enabled {
		maskinportenClient := client(app)

		*operations = append(*operations, resource.Operation{Resource: maskinportenClient, Operation: resource.OperationCreateOrUpdate})

		podSpec := &deployment.Spec.Template.Spec
		podSpec = pod.WithAdditionalSecret(podSpec, maskinportenClient.Spec.SecretName, nais.DefaultDigdiratorMaskinportenMountPath)
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, maskinportenClient.Spec.SecretName)
		deployment.Spec.Template.Spec = *podSpec
	}
	return
}
