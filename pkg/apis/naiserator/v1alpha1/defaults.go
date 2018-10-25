package v1alpha1

import (
	"github.com/imdario/mergo"
)

// Application spec default values
const (
	DefaultPortName = "http"
	DefaultPort     = 80
)

// ApplyDefaults sets default values where they are missing from an Application spec.
func ApplyDefaults(app *Application) error {
	return mergo.Merge(app, getAppDefaults())
}

func getAppDefaults() *Application {
	return &Application{
		Spec: ApplicationSpec{
			Replicas: Replicas{
				Min: 2,
				Max: 4,
				CpuThresholdPercentage: 50,
			},
			Port: 8080,
			Prometheus: PrometheusConfig{
				Enabled: false,
				Port:    "http",
				Path:    "/metrics",
			},
			Healthcheck: Healthcheck{
				Liveness: Probe{
					Path:             "isAlive",
					InitialDelay:     20,
					PeriodSeconds:    10,
					FailureThreshold: 3,
					Timeout:          1,
				},
				Readiness: Probe{
					Path:             "isReady",
					InitialDelay:     20,
					PeriodSeconds:    10,
					FailureThreshold: 3,
					Timeout:          1,
				},
			},
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
		}}
}
