package texas

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/observability"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const Port = 7164

type Source interface {
	resource.Source
}

type Config interface {
	GetClusterName() string
	GetObservability() config.Observability
	GetWebProxyOptions() config.Proxy
	IsGCPEnabled() bool
	IsTexasEnabled() bool
	TexasImage() string
}

func Create(
	source Source,
	ast *resource.Ast,
	cfg Config,
	clients Clients,
) error {
	if !cfg.IsTexasEnabled() || clients.IsEmpty() {
		return nil
	}

	if cfg.TexasImage() == "" {
		return fmt.Errorf("texas image not configured")
	}

	envs := clients.EnvVars()
	envs = append(envs, otelEnvVars(source, cfg)...)

	// If GCP is unconfigured, it means we are running on-premises, and we need the web proxy config.
	// Note that we need the web proxy regardless of whether the app has requested one, so we don't care about `.spec.webProxy`.
	if !cfg.IsGCPEnabled() {
		proxyEnvs, err := proxyopts.EnvironmentVariables(cfg)
		if err != nil {
			return fmt.Errorf("generate texas webproxy environment variables: %w", err)
		}
		envs = append(proxyEnvs, envs...)
	}

	ast.AppendEnv(applicationEnvVars()...)
	ast.InitContainers = append(ast.InitContainers, corev1.Container{
		Name:            "texas",
		Env:             envs,
		EnvFrom:         clients.EnvFromSources(),
		Image:           cfg.TexasImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    k8sResource.MustParse("20m"),
				corev1.ResourceMemory: k8sResource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: k8sResource.MustParse("256Mi"),
			},
		},
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways), // native sidecar
		SecurityContext: pod.DefaultContainerSecurityContext(),
	})
	ast.Labels["texas"] = "enabled"
	ast.Labels["otel"] = "enabled"

	return nil
}

type Clients struct {
	Azure        *nais_io_v1.AzureAdApplication
	IDPorten     *nais_io_v1.IDPortenClient
	Maskinporten *nais_io_v1.MaskinportenClient
	TokenX       *nais_io_v1.Jwker
}

func (c Clients) IsEmpty() bool {
	return c.Azure == nil && c.IDPorten == nil && c.Maskinporten == nil && c.TokenX == nil
}

func (c Clients) EnvVars() []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  "BIND_ADDRESS",
			Value: fmt.Sprintf("127.0.0.1:%d", Port),
		},
		{
			Name: "NAIS_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}
	if c.Azure != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "AZURE_ENABLED",
			Value: "true",
		})
	}
	if c.IDPorten != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "IDPORTEN_ENABLED",
			Value: "true",
		})
	}
	if c.Maskinporten != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "MASKINPORTEN_ENABLED",
			Value: "true",
		})
	}
	if c.TokenX != nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "TOKEN_X_ENABLED",
			Value: "true",
		})
	}
	return envs
}

func (c Clients) EnvFromSources() []corev1.EnvFromSource {
	sources := make([]corev1.EnvFromSource, 0)
	if c.Azure != nil {
		sources = append(sources, pod.EnvFromSecret(c.Azure.Spec.SecretName))
	}
	if c.IDPorten != nil {
		sources = append(sources, pod.EnvFromSecret(c.IDPorten.Spec.SecretName))
	}
	if c.Maskinporten != nil {
		sources = append(sources, pod.EnvFromSecret(c.Maskinporten.Spec.SecretName))
	}
	if c.TokenX != nil {
		sources = append(sources, pod.EnvFromSecret(c.TokenX.Spec.SecretName))
	}
	return sources
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

func otelEnvVars(source Source, cfg Config) []corev1.EnvVar {
	return observability.OtelEnvVars(
		"texas",
		source.GetNamespace(),
		[]corev1.EnvVar{
			{
				Name: "OTEL_RESOURCE_ATTRIBUTES",
				Value: fmt.Sprintf(
					"downstream.app.name=%s,downstream.app.namespace=%s,downstream.cluster.name=%s,nais.pod.name=$(NAIS_POD_NAME)",
					source.GetName(), source.GetNamespace(), cfg.GetClusterName(),
				),
			},
		},
		nil,
		cfg.GetObservability().Otel,
	)
}
