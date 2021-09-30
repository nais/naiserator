package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "http_requests",
		Namespace: "naiserator",
		Help:      "number of HTTP requests made to the health and liveness checks",
	})

	KubernetesResourceWriteDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:      "kubernetes_resource_write_duration",
		Namespace: "naiserator",
		Help:      "request duration when talking to Kubernetes",
		Buckets:   prometheus.LinearBuckets(0.01, 0.01, 100),
	})

	ResourcesGenerated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "resources_generated",
		Namespace: "naiserator",
		Help:      "number of Kubernetes resources that have been generated",
	}, []string{"kind"})

	ResourcesMonitored = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "resources_monitored",
		Namespace: "naiserator",
		Help:      "number of resources currently monitored for rollout completion",
	})

	Resources = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "resources",
		Namespace: "naiserator",
		Help:      "resources processed, with kind and status",
	}, []string{"kind", "status"})
)

func Register(registry prometheus.Registerer) {
	registry.MustRegister(
		Resources,
		ResourcesMonitored,
		ResourcesGenerated,
		HttpRequests,
		KubernetesResourceWriteDuration,
	)
}
