package jwker

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func jwkerSecretName(app nais_io_v1alpha1.Application) string {
	return namegen.PrefixedRandShortName("tokenx", app.Name, validation.DNS1035LabelMaxLength)
}

func Create(app *nais_io_v1alpha1.Application, options *resource.Options, operations *resource.Operations) {
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
	options.JwkerSecretName = jwker.Spec.SecretName
	return
}
