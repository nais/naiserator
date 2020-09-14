package resourcecreator

import (
	"fmt"
	idportenClient "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDPortenDefaultClientName            = "NAV"
	IDPortenDefaultClientURI             = "https://www.nav.no"
	IDPortenDefaultPostLogoutRedirectURI = "https://www.nav.no"
	IDPortenDefaultRefreshTokenLifetime  = 3600 * 12 // 12 hours, in seconds
)

func IDPortenClient(app nais.Application) (*idportenClient.IDPortenClient, error) {
	clientURI := app.Spec.IDPorten.ClientURI
	redirectURI := app.Spec.IDPorten.RedirectURI
	frontchannelLogoutURI := app.Spec.IDPorten.FrontchannelLogoutURI
	postLogoutRedirectURIs := app.Spec.IDPorten.PostLogoutRedirectURIs
	refreshTokenLifetime := app.Spec.IDPorten.RefreshTokenLifetime

	if len(clientURI) == 0 {
		clientURI = IDPortenDefaultClientURI
	}

	if len(app.Spec.Ingresses) == 0 {
		return nil, fmt.Errorf("you must specify an ingress to be able to use the idporten integration")
	}

	if len(redirectURI) == 0 {
		if len(app.Spec.Ingresses) == 1 {
			redirectURI = oauthCallbackURL(app.Spec.Ingresses[0])
		} else {
			return nil, fmt.Errorf("multiple ingresses exist with no redirect URI set for idporten, must be exactly one")
		}
	}

	if len(postLogoutRedirectURIs) == 0 {
		postLogoutRedirectURIs = []string{
			IDPortenDefaultPostLogoutRedirectURI,
		}
	}

	if refreshTokenLifetime == nil {
		lifetime := IDPortenDefaultRefreshTokenLifetime
		refreshTokenLifetime = &lifetime
	}

	return &idportenClient.IDPortenClient{
		TypeMeta: v1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: idportenClient.IDPortenClientSpec{
			ClientURI:              clientURI,
			RedirectURI:            redirectURI,
			SecretName:             getSecretName(app),
			FrontchannelLogoutURI:  frontchannelLogoutURI,
			PostLogoutRedirectURIs: postLogoutRedirectURIs,
			RefreshTokenLifetime:   *refreshTokenLifetime,
		},
	}, nil
}
