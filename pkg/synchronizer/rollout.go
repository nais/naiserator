package synchronizer

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/resourcecreator/resourceutils"
	appsv1 "k8s.io/api/apps/v1"
)

// Rollout represents the data neccessary to rollout an application to Kubernetes.
type Rollout struct {
	App                 *nais.Application
	ResourceOptions     resourceutils.Options
	ResourceOperations  resourcecreator.ResourceOperations
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
func (r *Rollout) SetCurrentDeployment(deployment *appsv1.Deployment) {
	if deployment != nil && deployment.Spec.Replicas != nil {
		r.ResourceOptions.NumReplicas = max(1, *deployment.Spec.Replicas)
	} else {
		r.ResourceOptions.NumReplicas = max(1, int32(r.App.Spec.Replicas.Min))
	}
}

func (r *Rollout) SetGoogleTeamProjectId(teamProjectId string) {
	r.ResourceOptions.GoogleTeamProjectId = teamProjectId
}
