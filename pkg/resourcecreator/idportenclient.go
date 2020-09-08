package resourcecreator

import (
	idportenClient "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IDPortenClient(application nais.Application) idportenClient.IDPortenClient {
	return idportenClient.IDPortenClient {
		TypeMeta: v1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: application.CreateObjectMeta(),
		Spec:       idportenClient.IDPortenClientSpec{
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