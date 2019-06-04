package fixtures

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Constant values for the variables returned in the Application spec.
const (
	ApplicationName      = "myapplication"
	ApplicationNamespace = "mynamespace"
	ApplicationTeam      = "myteam"
)

// MinimalApplication returns the absolute minimum application that might live in a Kubernetes cluster.
func MinimalFailingApplication() *nais.Application {
	return &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
		},
	}
}

// MinimalApplication returns the absolute minimum configuration needed to create a full set of Kubernetes resources.
func MinimalApplication() *nais.Application {
	return &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
			Labels: map[string]string{
				"team": ApplicationTeam,
			},
		},
	}
}
