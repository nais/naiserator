package wonderwall

import (
	"encoding/base64"
	"fmt"
	"strconv"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/keygen"
	"github.com/nais/liberator/pkg/namegen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/pointer"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
)

const (
	PortName        = "wonderwall"
	MetricsPortName = "ww-metrics"
	Port            = 7564
	MetricsPort     = 7565
	RedisSecretName = "redis-wonderwall"

	RedirectURIPath        = "/oauth2/callback"
	FrontChannelLogoutPath = "/oauth2/logout/frontchannel"
)

type Configuration struct {
	AutoLogin             bool
	ErrorPath             string
	Ingress               string
	Provider              string
	ProviderSecretName    string
	ACRValues             string
	UILocales             string
	PostLogoutRedirectURI string
}

func Create(app *nais_io_v1alpha1.Application, ast *resource.Ast, resourceOptions resource.Options, cfg Configuration) error {
	app.Labels["aiven"] = "enabled"

	if len(cfg.Provider) == 0 {
		return fmt.Errorf("configuration has empty provider")
	}

	if len(cfg.ProviderSecretName) == 0 {
		return fmt.Errorf("configuration has empty provider secret name")
	}

	if len(cfg.Ingress) == 0 {
		return fmt.Errorf("configuration has empty ingress")
	}

	wonderwallSecret, err := sidecarSecret(app, cfg)
	if err != nil {
		return err
	}

	container, err := sidecarContainer(app, resourceOptions, cfg)
	if err != nil {
		return err
	}

	container.EnvFrom = []corev1.EnvFromSource{
		pod.EnvFromSecret(cfg.ProviderSecretName),
		pod.EnvFromSecret(wonderwallSecret.GetName()),
		pod.EnvFromSecret(RedisSecretName),
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, wonderwallSecret)
	ast.Containers = append(ast.Containers, *container)

	return nil
}

func ShouldEnable(app *nais_io_v1alpha1.Application, resourceOptions resource.Options) bool {
	isGCP := len(resourceOptions.GoogleTeamProjectId) > 0

	if app.Spec.IDPorten == nil || app.Spec.IDPorten.Sidecar == nil || !isGCP {
		return false
	}

	idPortenEnabled := app.Spec.IDPorten.Enabled && app.Spec.IDPorten.Sidecar.Enabled

	return resourceOptions.DigdiratorEnabled && idPortenEnabled
}

func sidecarContainer(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, cfg Configuration) (*corev1.Container, error) {
	targetPort := app.Spec.Port
	resourcesSpec := nais_io_v1.ResourceRequirements{
		Limits: &nais_io_v1.ResourceSpec{
			Cpu:    "250m",
			Memory: "256Mi",
		},
		Requests: &nais_io_v1.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}

	return &corev1.Container{
		Name:            "wonderwall",
		Image:           resourceOptions.Wonderwall.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars(cfg, targetPort),
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
		Resources: pod.ResourceLimits(resourcesSpec),
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

func envVars(cfg Configuration, targetPort int) []corev1.EnvVar {
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
	return result
}

func sidecarSecret(source resource.Source, cfg Configuration) (*corev1.Secret, error) {
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
