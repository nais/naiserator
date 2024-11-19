package texas

import (
	"fmt"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/observability"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

type ProviderName string

const (
	ProviderAzureAD      ProviderName = "azure"
	ProviderIDPorten     ProviderName = "idporten"
	ProviderMaskinporten ProviderName = "maskinporten"
	ProviderTokenX       ProviderName = "tokenx"
)

const Port = 7164

const (
	// AnnotationEnable is an opt-in annotation key used to enable Texas sidecar
	// FIXME: remove when Texas is considered stable enough to be enabled by default
	AnnotationEnable = "texas.nais.io/enabled"
)

type Source interface {
	resource.Source
}

type Config interface {
	IsTexasEnabled() bool
	GetTexasOptions() config.Texas
	GetObservability() config.Observability
}

func Create(
	source Source,
	ast *resource.Ast,
	cfg Config,
	maskinportenclient *nais_io_v1.MaskinportenClient,
	azureadapplication *nais_io_v1.AzureAdApplication,
) error {
	needsTexas, ok := source.GetAnnotations()[AnnotationEnable]
	if !ok || needsTexas != "true" {
		return nil
	}

	if !cfg.IsTexasEnabled() {
		return fmt.Errorf("texas is not available in this cluster")
	}

	providers := NewProviders(maskinportenclient, azureadapplication)

	ast.AppendEnv(applicationEnvVars()...)
	ast.InitContainers = append(ast.InitContainers, sidecar(source, cfg, providers))
	ast.Labels["texas"] = "enabled"

	return nil
}

type Provider struct {
	Name       ProviderName
	SecretName string
}

func (p Provider) EnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  strings.ToUpper(string(p.Name)) + "_ENABLED",
		Value: "true",
	}
}

type Providers []Provider

func NewProviders(
	maskinportenclient *nais_io_v1.MaskinportenClient,
	azureadapplication *nais_io_v1.AzureAdApplication,
) Providers {
	providers := Providers{}

	if maskinportenclient != nil {
		providers = append(providers, Provider{
			Name:       ProviderMaskinporten,
			SecretName: maskinportenclient.Spec.SecretName,
		})
	}

	if azureadapplication != nil {
		providers = append(providers, Provider{
			Name:       ProviderAzureAD,
			SecretName: azureadapplication.Spec.SecretName,
		})
	}

	return providers
}

func (p Providers) EnvVars() []corev1.EnvVar {
	vars := make([]corev1.EnvVar, 0, len(p))
	for _, provider := range p {
		vars = append(vars, provider.EnvVar())
	}

	return vars
}

func (p Providers) EnvFromSources() []corev1.EnvFromSource {
	sources := make([]corev1.EnvFromSource, 0, len(p))
	for _, provider := range p {
		sources = append(sources, pod.EnvFromSecret(provider.SecretName))
	}

	return sources
}

func sidecar(source Source, cfg Config, providers Providers) corev1.Container {
	envs := []corev1.EnvVar{
		{
			Name:  "BIND_ADDRESS",
			Value: fmt.Sprintf("127.0.0.1:%d", Port),
		},
	}
	envs = append(envs, providers.EnvVars()...)
	envs = append(envs, observability.OtelEnvVars("texas", source.GetNamespace(), nil, nil, cfg.GetObservability().Otel)...)

	return corev1.Container{
		Name:            "texas",
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
		Image:           cfg.GetTexasOptions().Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envs,
		EnvFrom:         providers.EnvFromSources(),
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    k8sResource.MustParse("20m"),
				corev1.ResourceMemory: k8sResource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: k8sResource.MustParse("256Mi"),
			},
		},
		SecurityContext: pod.DefaultContainerSecurityContext(),
	}
}

// applicationEnvVars are the environment variables exposed to the main application container.
func applicationEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "NAIS_TOKEN_ENDPOINT",
			Value: fmt.Sprintf("http://127.0.0.1:%d/api/v1/token", Port),
		},
		{
			Name:  "NAIS_TOKEN_EXCHANGE_ENDPOINT",
			Value: fmt.Sprintf("http://127.0.0.1:%d/api/v1/token/exchange", Port),
		},
		{
			Name:  "NAIS_TOKEN_INTROSPECTION_ENDPOINT",
			Value: fmt.Sprintf("http://127.0.0.1:%d/api/v1/introspect", Port),
		},
	}
}
