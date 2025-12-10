package fixtures

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
) // Constant values for the variables returned in the Application spec.

const (
	ApplicationNamespace    = "mynamespace"
	DefaultApplicationName  = "myapplication"
	DefaultApplicationImage = "example/app:version"
	OtherApplicationName    = "otherapplication"
	OtherApplicationImage   = "example/other-image:version"
	OrphanedApplicationName = "orphanedapplication"
)

type FixtureModifier func(app client.Object)

func WithName(name string) FixtureModifier {
	return func(obj client.Object) {
		obj.SetName(name)
	}
}

func WithAnnotation(key, value string) FixtureModifier {
	return func(obj client.Object) {
		if obj.GetAnnotations() == nil {
			obj.SetAnnotations(make(map[string]string))
		}
		obj.GetAnnotations()[key] = value
	}
}

// MinimalApplication returns the absolute minimum configuration needed to create a full set of Kubernetes resources.
func MinimalApplication(modifiers ...FixtureModifier) *nais.Application {
	app := &nais.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: nais.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultApplicationName,
			Namespace: ApplicationNamespace,
		},
		Spec: nais.ApplicationSpec{
			Image: DefaultApplicationImage,
		},
	}
	err := app.ApplyDefaults()
	if err != nil {
		panic(err)
	}
	for _, modifier := range modifiers {
		modifier(app)
	}
	return app
}

func MinimalImage(modifiers ...FixtureModifier) *nais_io_v1.Image {
	img := &nais_io_v1.Image{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Image",
			APIVersion: nais_io_v1.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultApplicationName,
			Namespace: ApplicationNamespace,
		},
		Spec: nais_io_v1.ImageSpec{
			Image: OtherApplicationImage,
		},
	}
	for _, modifier := range modifiers {
		modifier(img)
	}
	return img
}
