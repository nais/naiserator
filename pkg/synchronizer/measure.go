package synchronizer

import (
	"time"

	"github.com/nais/naiserator/pkg/metrics"
)

// ObserveDuration measures the time used by a function, and reports it as a Kubernetes request duration metric.
func observeDuration(fun func() error) error {
	timer := time.Now()
	defer func() {
		used := time.Since(timer)
		metrics.KubernetesResourceWriteDuration.Observe(used.Seconds())
	}()
	return fun()
}
