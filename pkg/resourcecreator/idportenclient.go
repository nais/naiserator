package resourcecreator

import (
	"fmt"
	idportenClient "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	IDPortenDefaultClientURI             = "https://www.nav.no"
	IDPortenDefaultPostLogoutRedirectURI = "https://www.nav.no"
	IDPortenDefaultRefreshTokenLifetime  = 3600 * 12 // 12 hours, in seconds
)

func IDPortenClient(app *nais.Application) (*idportenClient.IDPortenClient, error) {
	if err := validateIngresses(*app); err != nil {
		return nil, err
	}

	setIDPortenDefaultsIfMissing(app)

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
			RefreshTokenLifetime:   *app.Spec.IDPorten.RefreshTokenLifetime,
		},
	}, nil
}

func setIDPortenDefaultsIfMissing(app *nais.Application) {
	if len(app.Spec.IDPorten.RedirectURI) == 0 {
		app.Spec.IDPorten.RedirectURI = oauthCallbackURL(app.Spec.Ingresses[0])
	}

	if len(app.Spec.IDPorten.ClientURI) == 0 {
		app.Spec.IDPorten.ClientURI = IDPortenDefaultClientURI
	}

	if len(app.Spec.IDPorten.PostLogoutRedirectURIs) == 0 {
		app.Spec.IDPorten.PostLogoutRedirectURIs = []string{IDPortenDefaultPostLogoutRedirectURI}
	}

	if app.Spec.IDPorten.RefreshTokenLifetime == nil {
		lifetime := IDPortenDefaultRefreshTokenLifetime
		app.Spec.IDPorten.RefreshTokenLifetime = &lifetime
	}
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

	if !strings.HasPrefix(redirectURI, ingress) {
		return fmt.Errorf("redirect URI ('%s') must be a subpath of the ingress ('%s')", redirectURI, ingress)
	}

	if !strings.HasPrefix(redirectURI, "https://") {
		return fmt.Errorf("redirect URI must start with https://")
	}
	return nil
}
