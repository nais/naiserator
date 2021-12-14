package synchronizer

import (
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

// Rollout represents the data necessary to rollout an application to Kubernetes.
type Rollout struct {
	Source              resource.Source
	ResourceOperations  resource.Operations
	CorrelationID       string
	SynchronizationHash string
}
