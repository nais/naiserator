package texas

import (
	"fmt"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

type ProviderName string

const (
	ProviderAzureAD      ProviderName = "azuread"
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
}

func Create(source Source, ast *resource.Ast, cfg Config, maskinportenClient *nais_io_v1.MaskinportenClient) error {
	needsTexas, ok := source.GetAnnotations()[AnnotationEnable]
	if !ok || needsTexas != "true" {
		return nil
	}

	if !cfg.IsTexasEnabled() {
		return fmt.Errorf("texas is not available in this cluster")
	}

	// FIXME: should probably be a constructor on Providers
	providers := Providers{
		{
			Name:       ProviderMaskinporten,
			SecretName: maskinportenClient.Spec.SecretName,
		},
	}

	ast.AppendEnv(applicationEnvVars()...)
	ast.InitContainers = append(ast.InitContainers, sidecar(cfg, providers))
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

func sidecar(cfg Config, providers Providers) corev1.Container {
	envs := []corev1.EnvVar{
		{
			Name:  "BIND_ADDRESS",
			Value: "127.0.0.1:1337",
		},
		// FIXME: otel envvars
	}
	envs = append(envs, providers.EnvVars()...)

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

		// FIXME: duplicated in securelogs/containers.go
		SecurityContext: &corev1.SecurityContext{
			Privileged:               ptr.To(false),
			AllowPrivilegeEscalation: ptr.To(false),
			ReadOnlyRootFilesystem:   ptr.To(true),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			RunAsUser:    ptr.To(int64(1069)),
			RunAsGroup:   ptr.To(int64(1069)),
			RunAsNonRoot: ptr.To(true),
		},
	}
}

// applicationEnvVars are the environment variables exposed to the main application container.
func applicationEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "AUTH_TOKEN_ENDPOINT",
			Value: fmt.Sprintf("http://127.0.0.1:%d/api/v1/token", Port),
		},
		{
			Name:  "AUTH_TOKEN_EXCHANGE_ENDPOINT",
			Value: fmt.Sprintf("http://127.0.0.1:%d/api/v1/token/exchange", Port),
		},
		{
			Name:  "AUTH_INTROSPECTION_ENDPOINT",
			Value: fmt.Sprintf("http://127.0.0.1:%d/api/v1/introspect", Port),
		},
	}
}
