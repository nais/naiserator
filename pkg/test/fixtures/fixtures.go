package fixtures

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Constant values for the variables returned in the Application spec.
const (
	ApplicationName      = "myapplication"
	ApplicationNamespace = "mynamespace"
	ApplicationTeam      = "myteam"
	NaisjobName          = "mynaisjob"
)

// MinimalApplication returns the absolute minimum application that might live in a Kubernetes cluster.
func MinimalFailingApplication() *nais.Application {
	app := &nais.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: nais.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
		},
	}
	err := app.ApplyDefaults()
	if err != nil {
		panic(err)
	}
	return app
}

// MinimalApplication returns the absolute minimum configuration needed to create a full set of Kubernetes resources.
func MinimalApplication() *nais.Application {
	app := &nais.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: nais.GroupVersion.Identifier(),
		},
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
	err := app.ApplyDefaults()
	if err != nil {
		panic(err)
	}
	return app
}

// MinimalNaisjob returns the absolute minimum configuration needed to create a full set of Kubernetes resources.
func MinimalNaisjob() *nais_io_v1.Naisjob {
	job := &nais_io_v1.Naisjob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Naisjob",
			APIVersion: nais.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      NaisjobName,
			Namespace: ApplicationNamespace,
			Labels: map[string]string{
				"team": ApplicationTeam,
			},
		},
		Spec: nais_io_v1.NaisjobSpec{
			Image: "example",
		},
	}
	err := job.ApplyDefaults()
	if err != nil {
		panic(err)
	}
	return job
}
