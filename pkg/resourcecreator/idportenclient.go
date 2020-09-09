package resourcecreator

import (
	idportenClient "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDPortenDefaultClientName = "NAV"
	IDPortenDefaultClientURI  = "https://www.nav.no"
)

func IDPortenClient(application nais.Application) idportenClient.IDPortenClient {
	if len(application.Spec.IDPorten.ClientName) == 0 {
		application.Spec.IDPorten.ClientName = IDPortenDefaultClientName
	}

	if len(application.Spec.IDPorten.ClientURI) == 0 {
		application.Spec.IDPorten.ClientURI = IDPortenDefaultClientURI
	}

	return idportenClient.IDPortenClient{
		TypeMeta: v1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: application.CreateObjectMeta(),
		Spec: idportenClient.IDPortenClientSpec{
			ClientName:             application.Spec.IDPorten.ClientName,
			ClientURI:              application.Spec.IDPorten.ClientURI,
			RedirectURIs:           application.Spec.IDPorten.RedirectURIs,
			SecretName:             getSecretName(application),
			FrontchannelLogoutURI:  application.Spec.IDPorten.FrontchannelLogoutURI,
			PostLogoutRedirectURIs: application.Spec.IDPorten.PostLogoutRedirectURIs,
			RefreshTokenLifetime:   application.Spec.IDPorten.RefreshTokenLifetime,
		},
	}
}
