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

func maskinPortenSecretName(appName string) string {
	return namegen.PrefixedRandShortName("maskinporten", appName, validation.DNS1035LabelMaxLength)
}

func client(objectMeta metav1.ObjectMeta, naisMaskinporten *nais_io_v1alpha1.Maskinporten) *nais_io_v1.MaskinportenClient {
	return &nais_io_v1.MaskinportenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MaskinportenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.MaskinportenClientSpec{
			Scopes:     naisMaskinporten.Scopes,
			SecretName: maskinPortenSecretName(objectMeta.Name),
		},
	}
}

func Create(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations, naisMaskinporten *nais_io_v1alpha1.Maskinporten) {
	if resourceOptions.DigdiratorEnabled && naisMaskinporten != nil && naisMaskinporten.Enabled {
		maskinportenClient := client(objectMeta, naisMaskinporten)

		*operations = append(*operations, resource.Operation{Resource: maskinportenClient, Operation: resource.OperationCreateOrUpdate})

		podSpec := &deployment.Spec.Template.Spec
		podSpec = pod.WithAdditionalSecret(podSpec, maskinportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, maskinportenClient.Spec.SecretName)
		deployment.Spec.Template.Spec = *podSpec
	}
	return
}
