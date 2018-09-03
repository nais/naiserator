package metrics

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Deployments = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "deployments",
		Namespace: "naiserator",
		Help: "Number of application deployments performed.",
	})
	ResourcesProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "resources_processed",
		Namespace: "naiserator",
		Help: "Number of nais.io.Application resources that have been processed.",
	})
	ResourcesGenerated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "resources_generated",
		Namespace: "naiserator",
		Help: "Number of Kubernetes resources that have been generated as a result of application deployments.",
	})
)

func init() {
	prometheus.MustRegister(Deployments)
	prometheus.MustRegister(ResourcesProcessed)
	prometheus.MustRegister(ResourcesGenerated)
}

// Serve Prometheus metric requests forever.
func Serve(addr, path string) {
	http.Handle(path, promhttp.Handler())
	glog.Infof("Serving metrics on %s path %s", addr, path)
	glog.Info(http.ListenAndServe(addr, nil))
}
