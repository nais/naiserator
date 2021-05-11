package jwker

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func jwkerSecretName(app nais_io_v1alpha1.Application) string {
	return namegen.PrefixedRandShortName("tokenx", app.Name, validation.DNS1035LabelMaxLength)
}

func Create(app *nais_io_v1alpha1.Application, options resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) {
	if !options.JwkerEnabled || !app.Spec.TokenX.Enabled {
		return
	}

	jwker := &nais_io_v1.Jwker{
		TypeMeta: v1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: nais_io_v1.JwkerSpec{
			AccessPolicy: accesspolicy.WithDefaults(app.Spec.AccessPolicy, app.Namespace, options.ClusterName),
			SecretName:   jwkerSecretName(*app),
		},
	}

	*operations = append(*operations, resource.Operation{jwker, resource.OperationCreateOrUpdate})

	podSpec := &deployment.Spec.Template.Spec
	podSpec = pod.WithAdditionalSecret(podSpec, jwker.Spec.SecretName, nais_io_v1alpha1.DefaultJwkerMountPath)
	if !app.Spec.TokenX.MountSecretsAsFilesOnly {
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, jwker.Spec.SecretName)
	}

	deployment.Spec.Template.Spec = *podSpec
	return
}
