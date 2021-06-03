package jwker

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func secretName(name string) string {
	return namegen.PrefixedRandShortName("tokenx", name, validation.DNS1035LabelMaxLength)
}

func Create(source resource.Source, ast *resource.Ast, options resource.Options, naisTokenX nais_io_v1.TokenX, naisAccessPolicy *nais_io_v1.AccessPolicy) {
	if !options.JwkerEnabled || !naisTokenX.Enabled {
		return
	}

	jwker := &nais_io_v1.Jwker{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: nais_io_v1.JwkerSpec{
			AccessPolicy: accesspolicy.WithDefaults(naisAccessPolicy, source.GetNamespace(), options.ClusterName),
			SecretName:   secretName(source.GetName()),
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, jwker)

	pod.WithAdditionalSecret(ast, jwker.Spec.SecretName, nais_io_v1alpha1.DefaultJwkerMountPath)
	if !naisTokenX.MountSecretsAsFilesOnly {
		pod.WithAdditionalEnvFromSecret(ast, jwker.Spec.SecretName)
	}
}
