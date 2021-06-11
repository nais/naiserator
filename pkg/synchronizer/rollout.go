package synchronizer

import (
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

// SetCurrentDeployment makes sure newly created Deployment objects matches autoscaling properties of an
// existing deployment.
//
// If the autoscaler is unavailable when a deployment is made, we risk scaling the application to the default
// number of replicas, which is set to one by default. To avoid this, we need to check the existing deployment
// resource and pass the correct number in the resource options.
//
// The number of replicas is set to whichever is highest: the current number of replicas (which might be zero),
// or the default number of replicas.
func (r *Rollout) SetCurrentDeployment(deployment *appsv1.Deployment, currentReplicasMin int) {
	if deployment != nil && deployment.Spec.Replicas != nil {
		r.ResourceOptions.NumReplicas = max(1, *deployment.Spec.Replicas)
	} else {
		r.ResourceOptions.NumReplicas = max(1, int32(currentReplicasMin))
	}
}

func (r *Rollout) SetGoogleTeamProjectId(teamProjectId string) {
	r.ResourceOptions.GoogleTeamProjectId = teamProjectId
}
