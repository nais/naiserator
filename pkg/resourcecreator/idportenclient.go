package resourcecreator

import (
	"fmt"
	"strings"

	idportenClient "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IDPortenClient(app *nais.Application) (*idportenClient.IDPortenClient, error) {
	if err := validateIngresses(*app); err != nil {
		return nil, err
	}

	if len(app.Spec.IDPorten.RedirectURI) == 0 {
		app.Spec.IDPorten.RedirectURI = oauthCallbackURL(app.Spec.Ingresses[0])
	}

	if len(app.Spec.IDPorten.PostLogoutRedirectURIs) == 0 {
		app.Spec.IDPorten.PostLogoutRedirectURIs = []string{}
	}

	if err := validateRedirectURI(*app); err != nil {
		return nil, err
	}

	return &idportenClient.IDPortenClient{
		TypeMeta: v1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: idportenClient.IDPortenClientSpec{
			ClientURI:              app.Spec.IDPorten.ClientURI,
			RedirectURI:            app.Spec.IDPorten.RedirectURI,
			SecretName:             getSecretName(*app),
			FrontchannelLogoutURI:  app.Spec.IDPorten.FrontchannelLogoutURI,
			PostLogoutRedirectURIs: app.Spec.IDPorten.PostLogoutRedirectURIs,
			SessionLifetime:        app.Spec.IDPorten.SessionLifetime,
			AccessTokenLifetime:    app.Spec.IDPorten.AccessTokenLifetime,
		},
	}, nil
}

func validateIngresses(app nais.Application) error {
	if len(app.Spec.Ingresses) == 0 {
		return fmt.Errorf("you must specify an ingress to be able to use the idporten integration")
	}

	if len(app.Spec.Ingresses) > 1 {
		return fmt.Errorf("cannot have more than one ingress when using the idporten integration")
	}
	return nil
}

func validateRedirectURI(app nais.Application) error {
	ingress := app.Spec.Ingresses[0]
	redirectURI := app.Spec.IDPorten.RedirectURI

	if !strings.HasPrefix(redirectURI, string(ingress)) {
		return fmt.Errorf("redirect URI ('%s') must be a subpath of the ingress ('%s')", redirectURI, ingress)
	}

	if !strings.HasPrefix(redirectURI, "https://") {
		return fmt.Errorf("redirect URI must start with https://")
	}
	return nil
}
