package resourcecreator

import (
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

func IDPortenClient(app nais.Application) idportenClient.IDPortenClient {
	clientName := app.Spec.IDPorten.ClientName
	clientURI := app.Spec.IDPorten.ClientURI
	redirectURIs := app.Spec.IDPorten.RedirectURIs
	frontchannelLogoutURI := app.Spec.IDPorten.FrontchannelLogoutURI
	postLogoutRedirectURIs := app.Spec.IDPorten.PostLogoutRedirectURIs
	refreshTokenLifetime := app.Spec.IDPorten.RefreshTokenLifetime

	if len(clientName) == 0 {
		clientName = IDPortenDefaultClientName
	}

	if len(clientURI) == 0 {
		clientURI = IDPortenDefaultClientURI
	}

	if len(redirectURIs) == 0 {
		redirectURIs = oauthCallbackURLs(app.Spec.Ingresses)
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

	return idportenClient.IDPortenClient{
		TypeMeta: v1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: idportenClient.IDPortenClientSpec{
			ClientName:             clientName,
			ClientURI:              clientURI,
			RedirectURIs:           redirectURIs,
			SecretName:             getSecretName(app),
			FrontchannelLogoutURI:  frontchannelLogoutURI,
			PostLogoutRedirectURIs: postLogoutRedirectURIs,
			RefreshTokenLifetime:   *refreshTokenLifetime,
		},
	}
}
