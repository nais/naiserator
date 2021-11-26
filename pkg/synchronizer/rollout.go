package synchronizer

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
)

// Rollout represents the data necessary to rollout an application to Kubernetes.
type Rollout struct {
	Source              resource.Source
	ResourceOptions     resource.Options
	ResourceOperations  resource.Operations
	CorrelationID       string
	SynchronizationHash string
}

type ReplicaResource interface {
	GetReplicas() *nais_io_v1.Replicas
}

// SetNumReplicas makes sure newly created Deployment objects matches autoscaling properties of an
// existing deployment.
//
// If the autoscaler is unavailable when a deployment is made, we risk scaling the application to the default
// number of replicas, which is set to one by default. To avoid this, we need to check the existing deployment
// resource and pass the correct number in the resource options.
//
// The number of replicas is set to whichever is highest: the current number of replicas (which might be zero),
// or the default number of replicas.
func (r *Rollout) SetNumReplicas(deployment *appsv1.Deployment, app ReplicaResource) {
	if *app.GetReplicas().Min == 0 && *app.GetReplicas().Max == 0 {
		// first, check if an app _should_ be scaled to zero by setting min = max = 0
		r.ResourceOptions.NumReplicas = 0
	} else if deployment != nil && deployment.Spec.Replicas != nil {
		// if a deployment already exists, use that deployment's number of replicas,
		// unless the minimum allowed replica count is below that of the application spec.
		r.ResourceOptions.NumReplicas = max(int32(*app.GetReplicas().Min), *deployment.Spec.Replicas)
	} else {
		// if this is a new deployment, fall back to the lowest number of replicas allowed in the application spec.
		r.ResourceOptions.NumReplicas = int32(*app.GetReplicas().Min)
	}
}

func (r *Rollout) SetGoogleTeamProjectId(teamProjectId string) {
	r.ResourceOptions.GoogleTeamProjectId = teamProjectId
}
