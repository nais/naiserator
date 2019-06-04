package fixtures

import (
	"strconv"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Constant values for the variables returned in the Application spec.
const (
	Name                      = "app"
	Namespace                 = "default"
	ImageName                 = "user/image:version"
	TeamName                  = "pandas"
	MinReplicas               = 1
	MaxReplicas               = 2
	CpuThresholdPercentage    = 69
	ReadinessPath             = "isready"
	ReadinessInitialDelay     = 1
	ReadinessTimeout          = 2
	ReadinessFailureThreshold = 3
	ReadinessPeriodSeconds    = 4
	LivenessPath              = "isalive"
	LivenessInitialDelay      = 5
	LivenessTimeout           = 6
	LivenessFailureThreshold  = 7
	LivenessPeriodSeconds     = 8
	RequestCpu                = "200m"
	RequestMemory             = "256Mi"
	LimitCpu                  = "500m"
	LimitMemory               = "512Mi"
	PrometheusPath            = "metrics"
	PreStopHookPath           = "die"
	LogFormat                 = "accesslog"
	LogTransform              = "dns_loglevel"
	VarName1                  = "varname1"
	VarValue1                 = "varvalue1"
	VarName2                  = "varname2"
	VarValue2                 = "varvalue2"
)

const ApplicationName = "myapplication"
const ApplicationNamespace = "mynamespace"
const ApplicationTeam = "myteam"

// Application returns a nais.io.Application test fixture.
func Application() *nais.Application {
	app := &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: Namespace,
			Labels: map[string]string{
				"team": TeamName,
			},
		},
		Spec: nais.ApplicationSpec{
			Port:  nais.DefaultAppPort,
			Image: ImageName,
			Strategy: nais.Strategy{
				Type: nais.DeploymentStrategyRollingUpdate,
			},
			Replicas: nais.Replicas{
				Min:                    MinReplicas,
				Max:                    MaxReplicas,
				CpuThresholdPercentage: CpuThresholdPercentage,
			},
			Readiness: nais.Probe{
				Path:             ReadinessPath,
				Port:             nais.DefaultAppPort,
				InitialDelay:     ReadinessInitialDelay,
				FailureThreshold: ReadinessFailureThreshold,
				Timeout:          ReadinessTimeout,
				PeriodSeconds:    ReadinessPeriodSeconds,
			},
			Liveness: nais.Probe{
				Path:             LivenessPath,
				Port:             nais.DefaultAppPort,
				InitialDelay:     LivenessInitialDelay,
				FailureThreshold: LivenessFailureThreshold,
				Timeout:          LivenessTimeout,
				PeriodSeconds:    LivenessPeriodSeconds,
			},
			Resources: nais.ResourceRequirements{
				Requests: nais.ResourceSpec{
					Memory: RequestMemory,
					Cpu:    RequestCpu,
				},
				Limits: nais.ResourceSpec{
					Memory: LimitMemory,
					Cpu:    LimitCpu,
				},
			},
			Prometheus: nais.PrometheusConfig{
				Path:    PrometheusPath,
				Port:    strconv.Itoa(nais.DefaultAppPort),
				Enabled: true,
			},
			Logtransform:    LogTransform,
			Logformat:       LogFormat,
			WebProxy:        true,
			PreStopHookPath: PreStopHookPath,
			LeaderElection:  true,
			Vault: nais.Vault{
				Enabled: false,
			},
			Env: []nais.EnvVar{
				{
					Name:  VarName1,
					Value: VarValue1,
				},
				{
					Name:  VarName2,
					Value: VarValue2,
				}},
			Service: nais.Service{
				Port: nais.DefaultServicePort,
			},
			AccessPolicy: nais.AccessPolicy{
				Ingress: nais.AccessPolicyIngress{
					AllowAll: false,
					Rules: []nais.AccessPolicyGressRule{
					},
				},
				Egress: nais.AccessPolicyEgress{
					AllowAll: false,
					Rules: []nais.AccessPolicyGressRule{
					},
				},
			},
		},
	}

	return app
}

// MinimalApplication returns the absolute minimum application that might live in a Kubernetes cluster.
func MinimalFailingApplication() *nais.Application {
	return &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
		},
	}
}

// MinimalApplication returns the absolute minimum configuration needed to create a full set of Kubernetes resources.
func MinimalApplication() *nais.Application {
	return &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
			Labels: map[string]string{
				"team": ApplicationTeam,
			},
		},
	}
}
