package resourcecreator

import (
	"fmt"
	"strings"

	idportenClient "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDPortenClientDefaultCallbackPath = "/oauth2/callback"
	IDPortenClientDefaultLogoutPath   = "/oauth2/logout"
)

func IDPortenClient(app *nais.Application) (*idportenClient.IDPortenClient, error) {
	if err := validateIngresses(app); err != nil {
		return nil, err
	}

	if err := validateRedirectURI(app); err != nil {
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
			RedirectURI:            redirectURI(app),
			SecretName:             getSecretName(*app),
			FrontchannelLogoutURI:  frontchannelLogoutURI(app),
			PostLogoutRedirectURIs: postLogoutRedirectURIs(app),
			SessionLifetime:        app.Spec.IDPorten.SessionLifetime,
			AccessTokenLifetime:    app.Spec.IDPorten.AccessTokenLifetime,
		},
	}, nil
}

func validateIngresses(app *nais.Application) error {
	if len(app.Spec.Ingresses) == 0 {
		return fmt.Errorf("you must specify an ingress to be able to use the idporten integration")
	}

	if len(app.Spec.Ingresses) > 1 {
		return fmt.Errorf("cannot have more than one ingress when using the idporten integration")
	}
	return nil
}

func validateRedirectURI(app *nais.Application) error {
	ingress := app.Spec.Ingresses[0]
	redirectURI := app.Spec.IDPorten.RedirectURI

	if len(redirectURI) == 0 {
		return nil
	}

	if !strings.HasPrefix(redirectURI, string(ingress)) {
		return fmt.Errorf("redirect URI ('%s') must be a subpath of the ingress ('%s')", redirectURI, ingress)
	}

	if !strings.HasPrefix(redirectURI, "https://") {
		return fmt.Errorf("redirect URI must start with https://")
	}
	return nil
}

func redirectURI(app *nais.Application) (redirectURI string) {
	redirectURI = app.Spec.IDPorten.RedirectURI

	if len(app.Spec.IDPorten.RedirectURI) == 0 {
		redirectURI = appendPathToIngress(app.Spec.Ingresses[0], IDPortenClientDefaultCallbackPath)
	}

	if len(app.Spec.IDPorten.RedirectPath) > 0 {
		redirectURI = appendPathToIngress(app.Spec.Ingresses[0], app.Spec.IDPorten.RedirectPath)
	}

	return
}

func frontchannelLogoutURI(app *nais.Application) (frontchannelLogoutURI string) {
	frontchannelLogoutURI = app.Spec.IDPorten.FrontchannelLogoutURI

	if len(app.Spec.IDPorten.FrontchannelLogoutURI) == 0 {
		frontchannelLogoutURI = appendPathToIngress(app.Spec.Ingresses[0], IDPortenClientDefaultLogoutPath)
	}

	if len(app.Spec.IDPorten.FrontchannelLogoutPath) > 0 {
		frontchannelLogoutURI = appendPathToIngress(app.Spec.Ingresses[0], app.Spec.IDPorten.FrontchannelLogoutPath)
	}

	return
}

func postLogoutRedirectURIs(app *nais.Application) (postLogoutRedirectURIs []string) {
	postLogoutRedirectURIs = app.Spec.IDPorten.PostLogoutRedirectURIs

	if len(app.Spec.IDPorten.PostLogoutRedirectURIs) == 0 {
		postLogoutRedirectURIs = []string{}
	}

	return
}
