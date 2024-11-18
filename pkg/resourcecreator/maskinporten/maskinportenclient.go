package maskinporten

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	// TexasEnableAnnotation is an opt-in annotation key used to enable Texas sidecar
	// FIXME: remove when Texas is considered stable enough to be enabled by default
	TexasEnableAnnotation = "texas.nais.io/enabled"
)

type Source interface {
	resource.Source
	GetMaskinporten() *nais_io_v1.Maskinporten
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

	enableTexas, ok := source.GetAnnotations()[TexasEnableAnnotation]
	if ok && enableTexas == "true" {
		if !cfg.IsTexasEnabled() {
			return fmt.Errorf("texas is not available in this cluster")
		}
		// FIXME: extract to common texas module
		{
			ast.AppendEnv(
				corev1.EnvVar{
					Name:  "AUTH_TOKEN_ENDPOINT",
					Value: "http://127.0.0.1:1337/api/v1/token",
				},
				corev1.EnvVar{
					Name:  "AUTH_TOKEN_EXCHANGE_ENDPOINT",
					Value: "http://127.0.0.1:1337/api/v1/token/exchange",
				},
				corev1.EnvVar{
					Name:  "AUTH_INTROSPECTION_ENDPOINT",
					Value: "http://127.0.0.1:1337/api/v1/introspect",
				},
			)
			ast.Labels["texas"] = "enabled"
		}

		ast.InitContainers = append(ast.InitContainers, texasSidecar(cfg, []string{maskinportenClient.Spec.SecretName}, []string{"maskinporten"}))
	}

	pod.WithAdditionalSecret(ast, maskinportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorMaskinportenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, maskinportenClient.Spec.SecretName)
	return nil
}
