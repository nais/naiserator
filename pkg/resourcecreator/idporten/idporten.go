package idporten

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/wonderwall"
	"github.com/nais/naiserator/pkg/util"
)

const (
	clientDefaultCallbackPath = "/oauth2/callback"
	clientDefaultLogoutPath   = "/oauth2/logout"
)

type Source interface {
	resource.Source
	wonderwall.Source
	GetIDPorten() *nais_io_v1.IDPorten
	GetIngress() []nais_io_v1.Ingress
}

type Config interface {
	wonderwall.Config
	IsWonderwallEnabled() bool
	IsDigdiratorEnabled() bool
}

func client(objectMeta metav1.ObjectMeta, naisIdPorten *nais_io_v1.IDPorten, naisIngresses []nais_io_v1.Ingress) (*nais_io_v1.IDPortenClient, error) {
	if err := validateIngresses(naisIngresses); err != nil {
		return nil, err
	}

	return &nais_io_v1.IDPortenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.IDPortenClientSpec{
			ClientURI:              naisIdPorten.ClientURI,
			RedirectURI:            redirectURI(naisIdPorten, naisIngresses),
			SecretName:             secretName(objectMeta.Name),
			FrontchannelLogoutURI:  frontchannelLogoutURI(naisIdPorten, naisIngresses),
			PostLogoutRedirectURIs: postLogoutRedirectURIs(naisIdPorten),
			SessionLifetime:        naisIdPorten.SessionLifetime,
			AccessTokenLifetime:    naisIdPorten.AccessTokenLifetime,
		},
	}, nil
}

func validateIngresses(ingresses []nais_io_v1.Ingress) error {
	if len(ingresses) == 0 {
		return fmt.Errorf("you must specify an ingress to be able to use the idporten integration")
	}

	if len(ingresses) > 1 {
		return fmt.Errorf("cannot have more than one ingress when using the idporten integration")
	}
	return nil
}

func redirectURI(idPorten *nais_io_v1.IDPorten, ingresses []nais_io_v1.Ingress) nais_io_v1.IDPortenURI {
	if len(idPorten.RedirectPath) == 0 {
		return idportenURI(ingresses, clientDefaultCallbackPath)
	}

	return idportenURI(ingresses, idPorten.RedirectPath)
}

func frontchannelLogoutURI(idPorten *nais_io_v1.IDPorten, ingresses []nais_io_v1.Ingress) nais_io_v1.IDPortenURI {
	if len(idPorten.FrontchannelLogoutPath) == 0 {
		return idportenURI(ingresses, clientDefaultLogoutPath)
	}

	return idportenURI(ingresses, idPorten.FrontchannelLogoutPath)
}

func postLogoutRedirectURIs(idPorten *nais_io_v1.IDPorten) (postLogoutRedirectURIs []nais_io_v1.IDPortenURI) {
	postLogoutRedirectURIs = idPorten.PostLogoutRedirectURIs

	if len(idPorten.PostLogoutRedirectURIs) == 0 {
		postLogoutRedirectURIs = make([]nais_io_v1.IDPortenURI, 0)
	}

	return
}

func idportenURI(ingresses []nais_io_v1.Ingress, path string) nais_io_v1.IDPortenURI {
	return nais_io_v1.IDPortenURI(util.AppendPathToIngress(ingresses[0], path))
}

func idportenURIs(ingresses []nais_io_v1.Ingress, path string) []nais_io_v1.IDPortenURI {
	uris := make([]nais_io_v1.IDPortenURI, 0)
	for _, ingress := range ingresses {
		uris = append(uris, nais_io_v1.IDPortenURI(util.AppendPathToIngress(ingress, path)))
	}

	return uris
}

func secretName(name string) string {
	return namegen.PrefixedRandShortName("idporten", name, validation.DNS1035LabelMaxLength)
}

func Create(app Source, ast *resource.Ast, cfg Config) error {
	idPorten := app.GetIDPorten()
	ingresses := app.GetIngress()

	if !cfg.IsDigdiratorEnabled() || idPorten == nil || !idPorten.Enabled {
		return nil
	}

	// create idporten client and attach secrets
	idportenClient, err := client(resource.CreateObjectMeta(app), idPorten, ingresses)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, idportenClient)

	pod.WithAdditionalSecret(ast, idportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, idportenClient.Spec.SecretName)

	// Return early if sidecar or Wonderwall is disabled
	if idPorten.Sidecar == nil || !idPorten.Sidecar.Enabled || !cfg.IsWonderwallEnabled() {
		return nil
	}

	// create sidecar container
	wonderwallCfg := wonderwallConfig(app, idportenClient.Spec.SecretName)
	err = wonderwall.Create(app, ast, cfg, wonderwallCfg)
	if err != nil {
		return err
	}

	// override uris when sidecar is enabled
	idportenClient.Spec.FrontchannelLogoutURI = idportenURI(ingresses, wonderwall.FrontChannelLogoutPath)
	idportenClient.Spec.RedirectURI = idportenURI(ingresses, wonderwall.RedirectURIPath)
	idportenClient.Spec.PostLogoutRedirectURIs = idportenURIs(ingresses, wonderwall.LogoutCallbackPath)

	return nil
}

func wonderwallConfig(source Source, providerSecretName string) wonderwall.Configuration {
	naisIngresses := source.GetIngress()
	naisIdPorten := source.GetIDPorten()

	cfg := wonderwall.Configuration{
		ACRValues:          naisIdPorten.Sidecar.Level,
		AutoLogin:          naisIdPorten.Sidecar.AutoLogin,
		ErrorPath:          naisIdPorten.Sidecar.ErrorPath,
		Ingress:            string(naisIngresses[0]),
		Loginstatus:        true,
		Provider:           "idporten",
		ProviderSecretName: providerSecretName,
		Resources:          naisIdPorten.Sidecar.Resources,
		UILocales:          naisIdPorten.Sidecar.Locale,
	}

	return cfg
}
