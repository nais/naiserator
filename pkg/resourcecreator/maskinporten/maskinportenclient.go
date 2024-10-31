package maskinporten

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Source interface {
	resource.Source
	GetMaskinporten() *nais_io_v1.Maskinporten
	GetTexas() *nais_io_v1.Texas
}

type Config interface {
	IsMaskinportenEnabled() bool
	IsTexasEnabled() bool
	GetTexasOptions() config.Texas
}

func secretName(name string) (string, error) {
	return namegen.ShortName(fmt.Sprintf("maskinporten-%s", name), validation.DNS1035LabelMaxLength)
}

func client(objectMeta metav1.ObjectMeta, naisMaskinporten *nais_io_v1.Maskinporten) (*nais_io_v1.MaskinportenClient, error) {
	name, err := secretName(objectMeta.Name)
	if err != nil {
		return nil, fmt.Errorf("generate secret name: %w", err)
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
			SecretName: name,
		},
	}, nil
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	maskinporten := source.GetMaskinporten()

	if maskinporten == nil || !maskinporten.Enabled {
		return nil
	}

	if !cfg.IsMaskinportenEnabled() {
		return fmt.Errorf("maskinporten is not available in this cluster")
	}

	ast.Labels["maskinporten"] = "enabled"

	maskinportenClient, err := client(resource.CreateObjectMeta(source), maskinporten)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, maskinportenClient)

	if cfg.IsTexasEnabled() {
		texas := source.GetTexas()
		if texas == nil || !texas.Maskinporten {
			return nil
		}
		// FIXME: only do this once
		{
			ast.AppendEnv(
				corev1.EnvVar{
					Name:  "TEXAS_TOKEN_ENDPOINT",
					Value: "http://127.0.0.1:1337/token",
				}, corev1.EnvVar{
					Name:  "TEXAS_INTROSPECTION_ENDPOINT",
					Value: "http://127.0.0.1:1337/introspection",
				},
			)
			ast.Labels["texas"] = "enabled"
		}

		ast.InitContainers = append(ast.InitContainers, texasSidecar(cfg, []string{maskinportenClient.Spec.SecretName}))
	} else {
		pod.WithAdditionalSecret(ast, maskinportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
		pod.WithAdditionalEnvFromSecret(ast, maskinportenClient.Spec.SecretName)
	}

	return nil
}
