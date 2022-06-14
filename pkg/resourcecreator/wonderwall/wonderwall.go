package wonderwall

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/imdario/mergo"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
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
	PortName        = "wonderwall"
	MetricsPortName = "ww-metrics"
	Port            = 7564
	MetricsPort     = 7565
	SecretName      = "wonderwall-secret"

	RedirectURIPath        = "/oauth2/callback"
	FrontChannelLogoutPath = "/oauth2/logout/frontchannel"
	LogoutCallbackPath     = "/oauth2/logout/callback"
)

type Configuration struct {
	ACRValues             string
	AutoLogin             bool
	ErrorPath             string
	Ingress               string
	Loginstatus           bool
	PostLogoutRedirectURI string
	Provider              string
	ProviderSecretName    string
	Resources             *nais_io_v1.ResourceRequirements
	UILocales             string
}

type Source interface {
	resource.Source
	GetPort() int
}

type Config interface {
	GetGoogleProjectID() string
	GetWonderwallOptions() config.Wonderwall
	IsDigdiratorEnabled() bool
	IsAzureratorEnabled() bool
}

func Create(source Source, ast *resource.Ast, config Config, cfg Configuration) error {
	ast.Labels["aiven"] = "enabled"
	ast.Labels["wonderwall"] = "enabled"

	if len(cfg.Provider) == 0 {
		return fmt.Errorf("configuration has empty provider")
	}

	if len(cfg.ProviderSecretName) == 0 {
		return fmt.Errorf("configuration has empty provider secret name")
	}

	if len(cfg.Ingress) == 0 {
		return fmt.Errorf("configuration has empty ingress")
	}

	wonderwallSecret, err := sidecarSecret(source, cfg)
	if err != nil {
		return err
	}

	container, err := sidecarContainer(source, config, cfg)
	if err != nil {
		return fmt.Errorf("creating wonderwall container spec: %w", err)
	}

	container.EnvFrom = []corev1.EnvFromSource{
		pod.EnvFromSecret(cfg.ProviderSecretName),
		pod.EnvFromSecret(wonderwallSecret.GetName()),
		pod.EnvFromSecret(SecretName),
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, wonderwallSecret)
	ast.Containers = append(ast.Containers, *container)

	return nil
}

func ShouldEnable(app *nais_io_v1alpha1.Application, opts Config) (bool, error) {
	if len(opts.GetGoogleProjectID()) == 0 {
		return false, nil
	}

	idPortenEnabled := opts.IsDigdiratorEnabled() &&
		app.Spec.IDPorten != nil &&
		app.Spec.IDPorten.Enabled &&
		app.Spec.IDPorten.Sidecar != nil &&
		app.Spec.IDPorten.Sidecar.Enabled

	azureEnabled := opts.IsAzureratorEnabled() &&
		app.Spec.Azure != nil &&
		app.Spec.Azure.Application != nil &&
		app.Spec.Azure.Application.Enabled &&
		app.Spec.Azure.Sidecar != nil &&
		app.Spec.Azure.Sidecar.Enabled

	if idPortenEnabled && azureEnabled {
		return false, fmt.Errorf("only one of Azure AD or ID-Porten sidecars can be enabled, but not both")
	}

	return idPortenEnabled || azureEnabled, nil
}

func sidecarContainer(source Source, config Config, cfg Configuration) (*corev1.Container, error) {
	targetPort := source.GetPort()
	options := config.GetWonderwallOptions()
	image := options.Image
	resourceReqs, err := resourceRequirements(cfg)
	if err != nil {
		return nil, err
	}

	return &corev1.Container{
		Name:            "wonderwall",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars(cfg, targetPort, options),
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
				Drop: []corev1.Capability{"all"},
			},
			Privileged:             pointer.Bool(false),
			ReadOnlyRootFilesystem: pointer.Bool(true),
			RunAsGroup:             pointer.Int64(1069),
			RunAsNonRoot:           pointer.Bool(true),
			RunAsUser:              pointer.Int64(1069),
		},
	}, nil
}

func resourceRequirements(cfg Configuration) (*nais_io_v1.ResourceRequirements, error) {
	reqs := cfg.Resources
	defaultReqs := &nais_io_v1.ResourceRequirements{
		Limits: &nais_io_v1.ResourceSpec{
			Cpu:    "250m",
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

func envVars(cfg Configuration, targetPort int, options config.Wonderwall) []corev1.EnvVar {
	result := []corev1.EnvVar{
		{
			Name:  "WONDERWALL_OPENID_PROVIDER",
			Value: cfg.Provider,
		},
		{
			Name:  "WONDERWALL_INGRESS",
			Value: cfg.Ingress,
		},
		{
			Name:  "WONDERWALL_UPSTREAM_HOST",
			Value: fmt.Sprintf("127.0.0.1:%d", targetPort),
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
	result = appendStringEnvVar(result, "WONDERWALL_ERROR_PATH", cfg.ErrorPath)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_ACR_VALUES", cfg.ACRValues)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_UI_LOCALES", cfg.UILocales)
	result = appendStringEnvVar(result, "WONDERWALL_OPENID_POST_LOGOUT_REDIRECT_URI", cfg.PostLogoutRedirectURI)

	if cfg.Loginstatus {
		result = appendBoolEnvVar(result, "WONDERWALL_LOGINSTATUS_ENABLED", options.Loginstatus.Enabled)
		result = appendStringEnvVar(result, "WONDERWALL_LOGINSTATUS_COOKIE_DOMAIN", options.Loginstatus.CookieDomain)
		result = appendStringEnvVar(result, "WONDERWALL_LOGINSTATUS_COOKIE_NAME", options.Loginstatus.CookieName)
		result = appendStringEnvVar(result, "WONDERWALL_LOGINSTATUS_RESOURCE_INDICATOR", options.Loginstatus.ResourceIndicator)
		result = appendStringEnvVar(result, "WONDERWALL_LOGINSTATUS_TOKEN_URL", options.Loginstatus.TokenURL)
	}

	return result
}

func sidecarSecret(source Source, cfg Configuration) (*corev1.Secret, error) {
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
