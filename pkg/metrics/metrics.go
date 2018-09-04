package metrics

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	ApplicationsSynchronized = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "applications_synchronized",
		Namespace: "naiserator",
		Help:      "number of nais.io.Application resources that have been synchronized",
	})
	ResourcesGenerated = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "resources_generated",
		Namespace: "naiserator",
		Help:      "number of Kubernetes resources that have been generated as a result of application deployments",
	})
)

func init() {
	prometheus.MustRegister(Deployments)
	prometheus.MustRegister(HttpRequests)
	prometheus.MustRegister(ApplicationsSynchronized)
	prometheus.MustRegister(ResourcesGenerated)
}

func isAlive(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Alive.")
	HttpRequests.Inc()
}

func isReady(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Ready.")
	HttpRequests.Inc()
}

// Serve health and metric requests forever.
func Serve(addr, metrics, ready, alive string) {
	h := http.NewServeMux()
	h.Handle(metrics, promhttp.Handler())
	h.HandleFunc(ready, isReady)
	h.HandleFunc(alive, isAlive)
	glog.Infof("HTTP server started on %s", addr)
	glog.Infof("Serving metrics on %s", metrics)
	glog.Infof("Serving readiness check on %s", ready)
	glog.Infof("Serving liveness check on %s", alive)
	glog.Info(http.ListenAndServe(addr, h))
}
