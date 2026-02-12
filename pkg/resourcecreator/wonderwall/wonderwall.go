package wonderwall

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/keygen"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/observability"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	Port                   = 7564
	MetricsPort            = 7565
	ProbePort              = 7566
	FrontChannelLogoutPath = "/oauth2/logout/frontchannel"
)

type Configuration struct {
	ACRValues             string
	AutoLogin             bool
	AutoLoginIgnorePaths  []nais_io_v1.WonderwallIgnorePaths
	NeedsEncryptionSecret bool
	Provider              string // should match values found in https://github.com/nais/wonderwall/blob/a5d98c746a4065b138bc01cf76005be0b6e1a5bb/pkg/config/openid.go#L10-L16
	Resources             *nais_io_v1.ResourceRequirements
	SecretNames           []string // secret name references to be mounted as env vars
	UILocales             string
}

type Source interface {
	resource.Source
	GetAzure() nais_io_v1.AzureInterface
	GetIDPorten() *nais_io_v1.IDPorten
	GetIngress() []nais_io_v1.Ingress
	GetLogin() *nais_io_v1.Login
	GetLiveness() *nais_io_v1.Probe
	GetPort() int
	GetPrometheus() *nais_io_v1.PrometheusConfig
	GetReadiness() *nais_io_v1.Probe
}

type Config interface {
	GetObservability() config.Observability
	GetWonderwallOptions() config.Wonderwall
	IsWonderwallEnabled() bool
}

func Create(source Source, ast *resource.Ast, naisCfg Config, wonderwallCfg Configuration) error {
	err := validate(source, naisCfg, wonderwallCfg)
	if err != nil {
		return err
	}

	container, err := sidecarContainer(source, naisCfg, wonderwallCfg)
	if err != nil {
		return fmt.Errorf("creating wonderwall container spec: %w", err)
	}

	if wonderwallCfg.NeedsEncryptionSecret {
		encryptionKeySecret, err := makeEncryptionKeySecret(source, wonderwallCfg)
		if err != nil {
			return err
		}

		ast.AppendOperation(resource.OperationCreateIfNotExists, encryptionKeySecret)
		container.EnvFrom = append(container.EnvFrom, pod.EnvFromSecret(encryptionKeySecret.GetName()))
	}

	ast.InitContainers = append(ast.InitContainers, *container)
	ast.Labels["aiven"] = "enabled"
	ast.Labels["otel"] = "enabled"
	ast.Labels["wonderwall"] = "enabled"
	return nil
}

func IsEnabled(source Source, config Config) bool {
	idporten := source.GetIDPorten()
	idPortenEnabled := idporten != nil && idporten.Sidecar != nil && idporten.Sidecar.Enabled

	azure := source.GetAzure()
	azureEnabled := azure != nil && azure.GetSidecar() != nil && azure.GetSidecar().Enabled

	login := source.GetLogin()
	loginEnabled := login != nil

	return config.IsWonderwallEnabled() && (idPortenEnabled || azureEnabled || loginEnabled)
}

func validate(source Source, naisCfg Config, wonderwallCfg Configuration) error {
	if !naisCfg.IsWonderwallEnabled() {
		return fmt.Errorf("wonderwall is not enabled for this cluster")
	}

	if naisCfg.GetWonderwallOptions().Image == "" {
		return fmt.Errorf("wonderwall image not configured")
	}

	if len(wonderwallCfg.Provider) == 0 {
		return fmt.Errorf("configuration has empty provider")
	}

	if len(wonderwallCfg.SecretNames) == 0 {
		return fmt.Errorf("configuration has no secret names")
	}

	if len(source.GetIngress()) == 0 {
		return fmt.Errorf("source has no ingresses")
	}

	for _, name := range wonderwallCfg.SecretNames {
		if len(name) == 0 {
			return fmt.Errorf("configuration contains empty secret names")
		}
	}

	idporten := source.GetIDPorten()
	idPortenEnabled := idporten != nil && idporten.Sidecar != nil && idporten.Sidecar.Enabled

	azure := source.GetAzure()
	azureEnabled := azure != nil && azure.GetSidecar() != nil && azure.GetSidecar().Enabled

	login := source.GetLogin()
	loginEnabled := login != nil

	if idPortenEnabled && azureEnabled || idPortenEnabled && loginEnabled || azureEnabled && loginEnabled {
		return fmt.Errorf("only one of Azure AD, ID-porten or login sidecars can be enabled")
	}

	port := source.GetPort()
	if port == Port || port == MetricsPort || port == ProbePort {
		return fmt.Errorf("cannot use port '%d'; conflicts with sidecar", port)
	}

	return nil
}

func sidecarContainer(source Source, naisCfg Config, wonderwallCfg Configuration) (*corev1.Container, error) {
	options := naisCfg.GetWonderwallOptions()
	image := options.Image
	resourceReqs, err := resourceRequirements(wonderwallCfg)
	if err != nil {
		return nil, err
	}

	envFromSources := make([]corev1.EnvFromSource, 0)
	for _, name := range wonderwallCfg.SecretNames {
		envFromSources = append(envFromSources, pod.EnvFromSecret(name))
	}

	return &corev1.Container{
		Name:            "wonderwall",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars(source, naisCfg, wonderwallCfg),
		EnvFrom:         envFromSources,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: int32(Port),
				Protocol:      corev1.ProtocolTCP,
				Name:          "wonderwall", // purely informational, should not be referenced by Service resources prior to v1.33
			},
			{
				ContainerPort: int32(MetricsPort),
				Protocol:      corev1.ProtocolTCP,
				Name:          "ww-metrics", // referenced by Prometheus PodMonitor
			},
			{
				ContainerPort: int32(ProbePort),
				Protocol:      corev1.ProtocolTCP,
				Name:          "ww-probe",
			},
		},
		Resources:     pod.ResourceLimits(*resourceReqs),
		RestartPolicy: new(corev1.ContainerRestartPolicyAlways),
		// StartupProbe ensures that the sidecar is ready to handle requests before the main application starts.
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt32(ProbePort),
				},
			},
		},
		SecurityContext: pod.DefaultContainerSecurityContext(),
	}, nil
}

func resourceRequirements(cfg Configuration) (*nais_io_v1.ResourceRequirements, error) {
	reqs := cfg.Resources
	defaultReqs := &nais_io_v1.ResourceRequirements{
		Limits: &nais_io_v1.ResourceSpec{
			Cpu:    "2",
			Memory: "256Mi",
		},
		Requests: &nais_io_v1.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}

	if reqs == nil {
		return defaultReqs, nil
	}

	err := mergo.Merge(reqs, defaultReqs)
	if err != nil {
		return nil, fmt.Errorf("merging default resource requirements: %w", err)
	}

	return reqs, nil
}

func envVars(source Source, naisCfg Config, cfg Configuration) []corev1.EnvVar {
	result := []corev1.EnvVar{
		{
			Name:  "WONDERWALL_OPENID_PROVIDER",
			Value: cfg.Provider,
		},
		{
			Name:  "WONDERWALL_INGRESS",
			Value: ingressString(source.GetIngress()),
		},
		{
			// Ideally we would be using the loopback address here, but some applications bind to _just_ eth0 instead of all interfaces
			Name: "WONDERWALL_UPSTREAM_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name:  "WONDERWALL_UPSTREAM_PORT",
			Value: fmt.Sprintf("%d", source.GetPort()),
		},
		{
			Name:  "WONDERWALL_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", Port),
		},
		{
			Name:  "WONDERWALL_METRICS_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", MetricsPort),
		},
		{
			Name:  "WONDERWALL_PROBE_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", ProbePort),
		},
	}

	result = appendBoolEnvVar(result, "WONDERWALL_AUTO_LOGIN", cfg.AutoLogin)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_ACR_VALUES", cfg.ACRValues)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_UI_LOCALES", cfg.UILocales)

	if cfg.AutoLogin {
		result = appendStringEnvVar(result, "WONDERWALL_AUTO_LOGIN_IGNORE_PATHS", autoLoginIgnorePaths(source, cfg))
	}

	otelEnvs := []corev1.EnvVar{
		{
			Name:  "OTEL_RESOURCE_ATTRIBUTES",
			Value: fmt.Sprintf("wonderwall.upstream.name=%s", source.GetName()),
		},
	}
	result = append(result, observability.OtelEnvVars("wonderwall", source.GetNamespace(), otelEnvs, nil, naisCfg.GetObservability().Otel)...)

	return result
}

func makeEncryptionKeySecret(source Source, cfg Configuration) (*corev1.Secret, error) {
	prefixedName := fmt.Sprintf("%s-wonderwall-%s", cfg.Provider, source.GetName())
	secretName, err := namegen.ShortName(prefixedName, validation.DNS1123LabelMaxLength)
	if err != nil {
		return nil, err
	}

	key, err := keygen.Keygen(32)
	if err != nil {
		return nil, fmt.Errorf("generating secret key: %w", err)
	}

	secrets := map[string]string{
		"WONDERWALL_ENCRYPTION_KEY": base64.StdEncoding.EncodeToString(key),
	}

	objectMeta := resource.CreateObjectMeta(source)
	sec := secret.OpaqueSecret(objectMeta, secretName, secrets)

	return sec, nil
}

func appendStringEnvVar(envVars []corev1.EnvVar, key, value string) []corev1.EnvVar {
	if len(value) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	return envVars
}

func appendBoolEnvVar(envVars []corev1.EnvVar, key string, value bool) []corev1.EnvVar {
	if value {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: strconv.FormatBool(value),
		})
	}

	return envVars
}

func autoLoginIgnorePaths(source Source, cfg Configuration) string {
	seen := make(map[string]bool)
	paths := make([]string, 0)

	addPath := func(p string) {
		if len(p) == 0 {
			return
		}

		p = leadingSlash(p)

		if _, found := seen[p]; !found {
			seen[p] = true
			paths = append(paths, p)
		}
	}

	if source.GetPrometheus() != nil && source.GetPrometheus().Enabled {
		addPath(source.GetPrometheus().Path)
	}

	if source.GetLiveness() != nil {
		addPath(source.GetLiveness().Path)
	}

	if source.GetReadiness() != nil {
		addPath(source.GetReadiness().Path)
	}

	for _, path := range cfg.AutoLoginIgnorePaths {
		addPath(string(path))
	}

	return strings.Join(paths, ",")
}

func leadingSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func ingressString(ingresses []nais_io_v1.Ingress) string {
	s := make([]string, len(ingresses))
	for i, ingress := range ingresses {
		s[i] = string(ingress)
	}
	return strings.Join(s, ",")
}
