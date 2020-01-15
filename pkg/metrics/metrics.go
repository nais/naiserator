package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	Deployments = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "deployments",
		Namespace: "naiserator",
		Help:      "number of application deployments performed",
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
	ResourcesGenerated = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "resources_generated",
		Namespace: "naiserator",
		Help:      "number of Kubernetes resources that have been generated as a result of application deployments",
	})
	QueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "queue_size",
		Namespace: "naiserator",
		Help:      "number of applications in processing queue",
	})
)

func init() {
	prometheus.MustRegister(Deployments)
	prometheus.MustRegister(HttpRequests)
	prometheus.MustRegister(ApplicationsProcessed)
	prometheus.MustRegister(ApplicationsFailed)
	prometheus.MustRegister(ResourcesGenerated)
	prometheus.MustRegister(QueueSize)
}

func isAlive(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "Alive.")
	if err != nil {
		log.Error("Failing when responding with Alive", err)
	}
	HttpRequests.Inc()
}

func isReady(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "Ready.")
	if err != nil {
		log.Error("Failing when responding with Ready", err)
	}
	HttpRequests.Inc()
}

// Serve health and metric requests forever.
func Serve(addr, metrics, ready, alive string) {
	h := http.NewServeMux()
	h.Handle(metrics, promhttp.Handler())
	h.HandleFunc(ready, isReady)
	h.HandleFunc(alive, isAlive)
	log.Infof("HTTP server started on %s", addr)
	log.Infof("Serving metrics on %s", metrics)
	log.Infof("Serving readiness check on %s", ready)
	log.Infof("Serving liveness check on %s", alive)
	log.Info(http.ListenAndServe(addr, h))
}
