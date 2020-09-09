package v1alpha1

import (
	"github.com/imdario/mergo"
)

// Application spec default values
const (
	DefaultPortName                 = "http"
	DefaultServicePort              = 80
	DefaultAppPort                  = 8080
	DefaultProbePeriodSeconds       = 10
	DefaultProbeTimeoutSeconds      = 1
	DefaultProbeFailureThreshold    = 3
	DeploymentStrategyRollingUpdate = "RollingUpdate"
	DeploymentStrategyRecreate      = "Recreate"
	DefaultVaultMountPath           = "/var/run/secrets/nais.io/vault"
)

// ApplyDefaults sets default values where they are missing from an Application spec.
func ApplyDefaults(app *Application) error {
	return mergo.Merge(app, getAppDefaults())
}

func getAppDefaults() *Application {
	return &Application{
		Spec: ApplicationSpec{
			Azure: &Azure{
				Application: &AzureApplication{
					Enabled: false,
				},
			},
			Replicas: &Replicas{
				Min:                    2,
				Max:                    4,
				CpuThresholdPercentage: 50,
			},
			Liveness: &Probe{
				PeriodSeconds:    DefaultProbePeriodSeconds,
				Timeout:          DefaultProbeTimeoutSeconds,
				FailureThreshold: DefaultProbeFailureThreshold,
			},
			Port: DefaultAppPort,
			Strategy: &Strategy{
				Type: DeploymentStrategyRollingUpdate,
			},
			Prometheus: &PrometheusConfig{
				Path: "/metrics",
			},
			Ingresses: []string{},
			Resources: &ResourceRequirements{
				Limits: &ResourceSpec{
					Cpu:    "500m",
					Memory: "512Mi",
				},
				Requests: &ResourceSpec{
					Cpu:    "200m",
					Memory: "256Mi",
				},
			},
			Vault: &Vault{
				Enabled: false,
				Paths:   []SecretPath{},
			},
			Service: &Service{
				Port: DefaultServicePort,
			},
			SecureLogs: &SecureLogs{
				Enabled: false,
			},
			AccessPolicy: &AccessPolicy{
				Inbound: &AccessPolicyInbound{
					Rules: []AccessPolicyRule{},
				},
				Outbound: &AccessPolicyOutbound{
					Rules:    []AccessPolicyRule{},
					External: []AccessPolicyExternalRule{},
				},
			},
			TokenX: &TokenX{
				Enabled:                 false,
				MountSecretsAsFilesOnly: false,
			},
			IDPorten: &IDPorten{
				Enabled:    false,
				ClientName: "NAV",
				ClientURI:  "https://www.nav.no",
			},
		},
	}
}
