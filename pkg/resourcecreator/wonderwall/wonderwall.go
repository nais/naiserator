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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/pointer"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
)

const (
	PortName               = "wonderwall"
	MetricsPortName        = "ww-metrics"
	Port                   = 7564
	MetricsPort            = 7565
	RedirectURIPath        = "/oauth2/callback"
	FrontChannelLogoutPath = "/oauth2/logout/frontchannel"
	LogoutCallbackPath     = "/oauth2/logout/callback"
)

type Configuration struct {
	ACRValues            string
	AutoLogin            bool
	AutoLoginIgnorePaths []nais_io_v1.WonderwallIgnorePaths
	Ingresses            []string
	Provider             string
	Resources            *nais_io_v1.ResourceRequirements
	SecretNames          []string
	UILocales            string
}

type Source interface {
	resource.Source
	GetAzure() nais_io_v1.AzureInterface
	GetIDPorten() *nais_io_v1.IDPorten
	GetLiveness() *nais_io_v1.Probe
	GetPort() int
	GetPrometheus() *nais_io_v1.PrometheusConfig
	GetReadiness() *nais_io_v1.Probe
}

type Config interface {
	GetWonderwallOptions() config.Wonderwall
	IsSeccompEnabled() bool
	IsWonderwallEnabled() bool
}

func Create(source Source, ast *resource.Ast, config Config, cfg Configuration) error {
	ast.Labels["aiven"] = "enabled"
	ast.Labels["wonderwall"] = "enabled"

	err := validate(source, config, cfg)
	if err != nil {
		return err
	}

	encryptionKeySecret, err := makeEncryptionKeySecret(source, cfg)
	if err != nil {
		return err
	}

	container, err := sidecarContainer(source, config, cfg, encryptionKeySecret)
	if err != nil {
		return fmt.Errorf("creating wonderwall container spec: %w", err)
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, encryptionKeySecret)
	ast.Containers = append(ast.Containers, *container)

	return nil
}

func validate(source Source, config Config, cfg Configuration) error {
	if !config.IsWonderwallEnabled() {
		return fmt.Errorf("wonderwall is not enabled for this cluster")
	}

	if len(cfg.Provider) == 0 {
		return fmt.Errorf("configuration has empty provider")
	}

	if len(cfg.SecretNames) == 0 {
		return fmt.Errorf("configuration has no secret names")
	}

	if len(cfg.Ingresses) == 0 {
		return fmt.Errorf("configuration has no ingresses")
	}

	for _, name := range cfg.SecretNames {
		if len(name) == 0 {
			return fmt.Errorf("configuration contains empty secret names")
		}
	}

	idporten := source.GetIDPorten()
	idPortenEnabled := idporten != nil && idporten.Sidecar != nil && idporten.Sidecar.Enabled

	azure := source.GetAzure()
	azureEnabled := azure != nil && azure.GetSidecar() != nil && azure.GetSidecar().Enabled

	if idPortenEnabled && azureEnabled {
		return fmt.Errorf("only one of Azure AD or ID-porten sidecars can be enabled, but not both")
	}

	return nil
}

func sidecarContainer(source Source, config Config, cfg Configuration, encryptionKeySecret *corev1.Secret) (*corev1.Container, error) {
	options := config.GetWonderwallOptions()
	image := options.Image
	resourceReqs, err := resourceRequirements(cfg)
	if err != nil {
		return nil, err
	}

	var sc *corev1.SeccompProfile
	if config.IsSeccompEnabled() {
		sc = &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		}
	}

	envFromSources := []corev1.EnvFromSource{
		pod.EnvFromSecret(encryptionKeySecret.GetName()),
	}
	for _, name := range cfg.SecretNames {
		envFromSources = append(envFromSources, pod.EnvFromSecret(name))
	}

	return &corev1.Container{
		Name:            "wonderwall",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars(source, cfg),
		EnvFrom:         envFromSources,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: int32(Port),
				Protocol:      corev1.ProtocolTCP,
				Name:          PortName,
			},
			{
				ContainerPort: int32(MetricsPort),
				Protocol:      corev1.ProtocolTCP,
				Name:          MetricsPortName,
			},
		},
		Resources: pod.ResourceLimits(*resourceReqs),
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: pointer.Bool(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			Privileged:             pointer.Bool(false),
			ReadOnlyRootFilesystem: pointer.Bool(true),
			RunAsGroup:             pointer.Int64(1069),
			RunAsNonRoot:           pointer.Bool(true),
			RunAsUser:              pointer.Int64(1069),
			SeccompProfile:         sc,
		},
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

func envVars(source Source, cfg Configuration) []corev1.EnvVar {
	result := []corev1.EnvVar{
		{
			Name:  "WONDERWALL_OPENID_PROVIDER",
			Value: cfg.Provider,
		},
		{
			Name:  "WONDERWALL_INGRESS",
			Value: strings.Join(cfg.Ingresses, ","),
		},
		{
			Name:  "WONDERWALL_UPSTREAM_HOST",
			Value: fmt.Sprintf("127.0.0.1:%d", source.GetPort()),
		},
		{
			Name:  "WONDERWALL_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", Port),
		},
		{
			Name:  "WONDERWALL_METRICS_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", MetricsPort),
		},
	}

	result = appendBoolEnvVar(result, "WONDERWALL_AUTO_LOGIN", cfg.AutoLogin)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_ACR_VALUES", cfg.ACRValues)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_UI_LOCALES", cfg.UILocales)

	if cfg.AutoLogin {
		result = appendStringEnvVar(result, "WONDERWALL_AUTO_LOGIN_IGNORE_PATHS", autoLoginIgnorePaths(source, cfg))
	}

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
