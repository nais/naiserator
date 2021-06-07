package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Deployments = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "deployments",
		Namespace: "naiserator",
		Help:      "number of application deployments performed",
	})
	NaisjobsDeployments = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "naisjob_deployments",
		Namespace: "naiserator",
		Help:      "number of naisjob deployments performed",
	})
	HttpRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "http_requests",
		Namespace: "naiserator",
		Help:      "number of HTTP requests made to the health and liveness checks",
	})
	ApplicationsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "applications_processed",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that have been processed",
	})
	ApplicationsFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "applications_failed",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that failed processing",
	})
	ApplicationsRetries = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "applications_retried",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that failed synchronization and have been re-enqueued",
	})
	NaisjobsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "naisjobs_processed",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that have been processed",
	})
	NaisjobsFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "naisjobs_failed",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that failed processing",
	})
	NaisjobsRetries = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "naisjobs_retried",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that failed synchronization and have been re-enqueued",
	})
	ResourcesGenerated = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "resources_generated",
		Namespace: "naiserator",
		Help:      "number of Kubernetes resources that have been generated as a result of application deployments",
	})
	KubernetesResourceWriteDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:      "kubernetes_resource_write_duration",
		Namespace: "naiserator",
		Help:      "request duration when talking to Kubernetes",
		Buckets:   prometheus.LinearBuckets(0.01, 0.01, 100),
	})
)

func Register(registry prometheus.Registerer) {
	registry.MustRegister(
		Deployments,
		NaisjobsDeployments,
		HttpRequests,
		ApplicationsProcessed,
		ApplicationsFailed,
		ApplicationsRetries,
		NaisjobsProcessed,
		NaisjobsFailed,
		NaisjobsRetries,
		ResourcesGenerated,
		KubernetesResourceWriteDuration,
	)
}
