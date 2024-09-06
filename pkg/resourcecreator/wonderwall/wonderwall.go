package wonderwall

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	FrontChannelLogoutPath = "/oauth2/logout/frontchannel"
)

type Configuration struct {
	ACRValues             string
	AutoLogin             bool
	AutoLoginIgnorePaths  []nais_io_v1.WonderwallIgnorePaths
	Ingresses             []nais_io_v1.Ingress
	NeedsEncryptionSecret bool
	Provider              string
	Resources             *nais_io_v1.ResourceRequirements
	SecretNames           []string
	UILocales             string
}

type Source interface {
	resource.Source
	GetAzure() nais_io_v1.AzureInterface
	GetIDPorten() *nais_io_v1.IDPorten
	GetLiveness() *nais_io_v1.Probe
	GetPort() int
	GetPrometheus() *nais_io_v1.PrometheusConfig
	GetReadiness() *nais_io_v1.Probe
	GetTerminationGracePeriodSeconds() *int64
}

type Config interface {
	GetWonderwallOptions() config.Wonderwall
	IsWonderwallEnabled() bool
}

func Create(source Source, ast *resource.Ast, naisCfg Config, wonderwallCfg Configuration) error {
	ast.Labels["aiven"] = "enabled"
	ast.Labels["wonderwall"] = "enabled"

	err := validate(source, naisCfg, wonderwallCfg)
	if err != nil {
		return err
	}

	container, err := sidecarContainer(source, naisCfg, wonderwallCfg)
	if err != nil {
		return fmt.Errorf("NAISERATOR-5263: creating wonderwall container spec: %w", err)
	}

	if wonderwallCfg.NeedsEncryptionSecret {
		encryptionKeySecret, err := makeEncryptionKeySecret(source, wonderwallCfg)
		if err != nil {
			return err
		}

		ast.AppendOperation(resource.OperationCreateIfNotExists, encryptionKeySecret)
		container.EnvFrom = append(container.EnvFrom, pod.EnvFromSecret(encryptionKeySecret.GetName()))
	}

	ast.Containers = append(ast.Containers, *container)

	return nil
}

func validate(source Source, naisCfg Config, wonderwallCfg Configuration) error {
	if !naisCfg.IsWonderwallEnabled() {
		return fmt.Errorf("NAISERATOR-2114: wonderwall is not enabled for this cluster")
	}

	if len(wonderwallCfg.Provider) == 0 {
		return fmt.Errorf("NAISERATOR-0820: configuration has empty provider")
	}

	if len(wonderwallCfg.SecretNames) == 0 {
		return fmt.Errorf("NAISERATOR-2265: configuration has no secret names")
	}

	if len(wonderwallCfg.Ingresses) == 0 {
		return fmt.Errorf("NAISERATOR-0110: configuration has no ingresses")
	}

	for _, name := range wonderwallCfg.SecretNames {
		if len(name) == 0 {
			return fmt.Errorf("NAISERATOR-4599: configuration contains empty secret names")
		}
	}

	idporten := source.GetIDPorten()
	idPortenEnabled := idporten != nil && idporten.Sidecar != nil && idporten.Sidecar.Enabled

	azure := source.GetAzure()
	azureEnabled := azure != nil && azure.GetSidecar() != nil && azure.GetSidecar().Enabled

	if idPortenEnabled && azureEnabled {
		return fmt.Errorf("NAISERATOR-1511: only one of Azure AD or ID-porten sidecars can be enabled, but not both")
	}

	port := source.GetPort()
	if port == Port || port == MetricsPort {
		return fmt.Errorf("NAISERATOR-7360: cannot use port '%d'; conflicts with sidecar", port)
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
		Env:             envVars(source, wonderwallCfg),
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
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
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
		return nil, fmt.Errorf("NAISERATOR-1549: merging default resource requirements: %w", err)
	}

	return reqs, nil
}

func envVars(source Source, cfg Configuration) []corev1.EnvVar {
	terminationGracePeriodSeconds := 30
	if source.GetTerminationGracePeriodSeconds() != nil {
		terminationGracePeriodSeconds = int(*source.GetTerminationGracePeriodSeconds())
	}

	result := []corev1.EnvVar{
		{
			Name:  "WONDERWALL_OPENID_PROVIDER",
			Value: cfg.Provider,
		},
		{
			Name:  "WONDERWALL_INGRESS",
			Value: ingressString(cfg.Ingresses),
		},
		{
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
			Name:  "WONDERWALL_SHUTDOWN_GRACEFUL_PERIOD",
			Value: fmt.Sprintf("%s", time.Duration(terminationGracePeriodSeconds)*time.Second),
		},
		{
			Name:  "WONDERWALL_SHUTDOWN_WAIT_BEFORE_PERIOD",
			Value: "7s", // should be less than linkerd's sleep (10s) and greater than application's sleep (5s)
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
		return nil, fmt.Errorf("NAISERATOR-7195: generating secret key: %w", err)
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
