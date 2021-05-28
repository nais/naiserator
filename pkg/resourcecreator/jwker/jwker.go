package jwker

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func jwkerSecretName(name string) string {
	return namegen.PrefixedRandShortName("tokenx", name, validation.DNS1035LabelMaxLength)
}

func Create(objectMeta metav1.ObjectMeta, options resource.Options, deployment *appsv1.Deployment, operations *resource.Operations, naisTokenX nais_io_v1alpha1.TokenX, naisAccessPolicy *nais_io_v1.AccessPolicy) {
	if !options.JwkerEnabled || !naisTokenX.Enabled {
		return
	}

	jwker := &nais_io_v1.Jwker{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.JwkerSpec{
			AccessPolicy: accesspolicy.WithDefaults(naisAccessPolicy, objectMeta.Namespace, options.ClusterName),
			SecretName:   jwkerSecretName(objectMeta.Name),
		},
	}

	*operations = append(*operations, resource.Operation{Resource: jwker, Operation: resource.OperationCreateOrUpdate})

	podSpec := &deployment.Spec.Template.Spec
	podSpec = pod.WithAdditionalSecret(podSpec, jwker.Spec.SecretName, nais_io_v1alpha1.DefaultJwkerMountPath)
	if !naisTokenX.MountSecretsAsFilesOnly {
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, jwker.Spec.SecretName)
	}

	deployment.Spec.Template.Spec = *podSpec
	return
}
