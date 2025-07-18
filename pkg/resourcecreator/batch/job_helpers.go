package batch

import v1 "k8s.io/api/core/v1"

func RestartPolicy(restartPolicy string) v1.RestartPolicy {
	defaultPolicy := v1.RestartPolicyNever
	if restartPolicy == string(v1.RestartPolicyOnFailure) {
		return v1.RestartPolicyOnFailure
	}
	return defaultPolicy
}
