package resourcecreator

import (
	"strings"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	Name                      = "app"
	Namespace                 = "default"
	Port                      = 8080
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
	PrometheusPort            = "http"
	PrometheusEnabled         = true
	IstioEnabled              = true
	WebProxyEnabled           = true
	IngressDisabled           = true
	LeaderElectionEnabled     = true
	SecretsEnabled            = true
	PreStopHookPath           = "die"
	LogFormat                 = "accesslog"
	LogTransform              = "dns_loglevel"
)

func TestCreateResourceSpecs(t *testing.T) {
	app := getExampleApp()

	specs, e := Create(app)
	assert.NoError(t, e)

	svc := get(specs, "service").(*v1.Service)
	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))

	deploy := get(specs, "deployment").(*appsv1.Deployment)
	assert.Equal(t, ImageName, deploy.Spec.Template.Spec.Containers[0].Image)
}

func get(resources []runtime.Object, kind string) runtime.Object {
	for _, r := range resources {
		if strings.EqualFold(r.GetObjectKind().GroupVersionKind().Kind, kind) {
			return r
		}
	}
	panic("no matching resource kind found")
}

func getExampleApp() *nais.Application {
	app := &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: Namespace,
		},
		Spec: nais.ApplicationSpec{
			Port:  Port,
			Image: ImageName,
			Team:  TeamName,
			Replicas: nais.Replicas{
				Min:                    MinReplicas,
				Max:                    MaxReplicas,
				CpuThresholdPercentage: CpuThresholdPercentage,
			},
			Healthcheck: nais.Healthcheck{
				Readiness: nais.Probe{
					Path:             ReadinessPath,
					InitialDelay:     ReadinessInitialDelay,
					FailureThreshold: ReadinessFailureThreshold,
					Timeout:          ReadinessTimeout,
					PeriodSeconds:    ReadinessPeriodSeconds,
				},
				Liveness: nais.Probe{
					Path:             LivenessPath,
					InitialDelay:     LivenessInitialDelay,
					FailureThreshold: LivenessFailureThreshold,
					Timeout:          LivenessTimeout,
					PeriodSeconds:    LivenessPeriodSeconds,
				},
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
				Port:    PrometheusPort,
				Enabled: PrometheusEnabled,
			},
			Istio: nais.IstioConfig{
				Enabled: IstioEnabled,
			},
			Logtransform:    LogTransform,
			Logformat:       LogFormat,
			WebProxy:        WebProxyEnabled,
			PreStopHookPath: PreStopHookPath,
			Ingress: nais.Ingress{
				Disabled: IngressDisabled,
			},
			LeaderElection: LeaderElectionEnabled,
			Secrets:        SecretsEnabled,
		}}

	return app
}
