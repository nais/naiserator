package synchronizer

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
)

// Rollout represents the data neccessary to rollout an application to Kubernetes.
type Rollout struct {
	App                nais.Application
	ResourceOptions    resourcecreator.ResourceOptions
	ResourceOperations resourcecreator.ResourceOperations
}
