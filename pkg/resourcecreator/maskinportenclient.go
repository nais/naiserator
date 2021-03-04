package resourcecreator

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MaskinportenClient(app *nais.Application) (*v1.MaskinportenClient, error) {
	secretName, err := util.GenerateSecretName("maskinporten", app.Name, MaxSecretNameLength)
	if err != nil {
		return nil, err
	}

	return &v1.MaskinportenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MaskinportenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: v1.MaskinportenClientSpec{
			Scopes:     app.Spec.Maskinporten.Scopes,
			SecretName: secretName,
		},
	}, nil
}
