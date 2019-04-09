package v1alpha1

import (
	"github.com/imdario/mergo"
	"strconv"
)

// Application spec default values
const (
	DefaultPortName                 = "http"
	DefaultPort                     = 80
	DefaultServicePort              = 8080
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
			Replicas: Replicas{
				Min:                    2,
				Max:                    4,
				CpuThresholdPercentage: 50,
			},
			Port: DefaultServicePort,
			Strategy: Strategy{
				Type: DeploymentStrategyRollingUpdate,
			},
			Prometheus: PrometheusConfig{
				Enabled: false,
				Port:    strconv.Itoa(DefaultServicePort),
				Path:    "/metrics",
			},
			Ingresses: []string{},
			Resources: ResourceRequirements{
				Limits: ResourceSpec{
					Cpu:    "500m",
					Memory: "512Mi",
				},
				Requests: ResourceSpec{
					Cpu:    "200m",
					Memory: "256Mi",
				},
			},
			Vault: Vault{
				Enabled: false,
				Mounts:  []SecretPath{},
			},
			Service: Service{
				Port: DefaultPort,
			},
		},
	}
}
