package podmonitor

import (
	"strconv"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	pov1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	resource.Source
	GetPrometheus() *nais_io_v1.PrometheusConfig
	GetPort() int
}

type Config interface {
	IsPrometheusOperatorEnabled() bool
}

func Create(source Source, ast *resource.Ast, config Config) {
	prom := source.GetPrometheus()
	if !prom.Enabled || !config.IsPrometheusOperatorEnabled() {
		return
	}

	port := nais_io_v1alpha1.DefaultPortName
	promPort := prom.Port
	if len(promPort) > 0 && promPort != strconv.Itoa(source.GetPort()) {
		port = "metrics"
	}

	podMonitor := &pov1.PodMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMonitor",
			APIVersion: pov1.SchemeGroupVersion.String(),
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: pov1.PodMonitorSpec{
			JobLabel: "app.kubernetes.io/name",
			PodMetricsEndpoints: []pov1.PodMetricsEndpoint{
				{
					Port:        port,
					Path:        prom.Path,
					HonorLabels: false,
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": source.GetName()},
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, podMonitor)
}
