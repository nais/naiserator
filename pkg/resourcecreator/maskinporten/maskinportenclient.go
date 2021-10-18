package maskinporten

import (
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func secretName(appName string) (string, error) {
	basename := fmt.Sprintf("%s-%s", "maskinporten", appName)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d", year, week)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(basename, suffix, maxLen)
}

func client(objectMeta metav1.ObjectMeta, naisMaskinporten *nais_io_v1.Maskinporten) (*nais_io_v1.MaskinportenClient, error) {
	secretName, err := secretName(objectMeta.Name)
	if err != nil {
		return nil, err
	}

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
			SecretName: secretName,
		},
	}, nil
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisMaskinporten *nais_io_v1.Maskinporten) error {
	if !resourceOptions.DigdiratorEnabled || naisMaskinporten == nil || !naisMaskinporten.Enabled {
		return nil
	}

	maskinportenClient, err := client(resource.CreateObjectMeta(source), naisMaskinporten)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, maskinportenClient)
	pod.WithAdditionalSecret(ast, maskinportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, maskinportenClient.Spec.SecretName)

	return nil
}
