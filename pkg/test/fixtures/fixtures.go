package fixtures

import (
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
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
	app := &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
		},
	}
	err := nais.ApplyDefaults(app)
	if err != nil {
		panic(err)
	}
	return app
}

// MinimalApplication returns the absolute minimum configuration needed to create a full set of Kubernetes resources.
func MinimalApplication() *nais.Application {
	app := &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
			Labels: map[string]string{
				"team": ApplicationTeam,
			},
		},
		Spec: nais.ApplicationSpec{
			Image: "example",
		},
	}
	err := nais.ApplyDefaults(app)
	if err != nil {
		panic(err)
	}
	return app
}
