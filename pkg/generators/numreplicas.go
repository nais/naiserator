package generators

import (
	appsv1 "k8s.io/api/apps/v1"
)

// numReplicas returns the number of replicas suitable for an update to an existing deployment.
//
// If the autoscaler is unavailable when a deployment is made, we risk scaling the application to the default
// number of replicas, which is set to one by default. To avoid this, we need to check the existing deployment
// resource and pass the correct number in the resource options.
//
// The number of replicas is set to whichever is highest: the current number of replicas (which might be zero),
// or the default number of replicas.
func numReplicas(deployment *appsv1.Deployment, minReplicas, maxReplicas *int) int32 {
	if *minReplicas == 0 && *maxReplicas == 0 {
		// first, check if an app _should_ be scaled to zero by setting min = max = 0
		return 0
	} else if deployment != nil && deployment.Spec.Replicas != nil {
		// if a deployment already exists, use that deployment's number of replicas,
		// unless the minimum allowed replica count is below that of the application spec.
		return max(int32(*minReplicas), *deployment.Spec.Replicas)
	} else {
		// if this is a new deployment, fall back to the lowest number of replicas allowed in the application spec.
		return int32(*minReplicas)
	}
}

// max returns the larger integer of a, b.
func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
