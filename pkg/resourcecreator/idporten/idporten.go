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
	wonderwallSecretName      = "wonderwall-idporten-config"
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
			IntegrationType:        naisIdPorten.IntegrationType,
			RedirectURIs:           redirectURIs(naisIdPorten, naisIngresses),
			SecretName:             secretName(objectMeta.Name),
			FrontchannelLogoutURI:  frontchannelLogoutURI(naisIdPorten, naisIngresses),
			PostLogoutRedirectURIs: postLogoutRedirectURIs(naisIdPorten),
			SessionLifetime:        naisIdPorten.SessionLifetime,
			AccessTokenLifetime:    naisIdPorten.AccessTokenLifetime,
			Scopes:                 naisIdPorten.Scopes,
		},
	}, nil
}

func validateIngresses(ingresses []nais_io_v1.Ingress) error {
	if len(ingresses) == 0 {
		return fmt.Errorf("you must specify an ingress to be able to use the idporten integration")
	}

	return nil
}

func redirectURIs(idPorten *nais_io_v1.IDPorten, ingresses []nais_io_v1.Ingress) []nais_io_v1.IDPortenURI {
	if len(idPorten.RedirectPath) == 0 {
		return idportenURIs(ingresses, clientDefaultCallbackPath)
	}

	return idportenURIs(ingresses, idPorten.RedirectPath)
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

	if idPorten == nil || !idPorten.Enabled {
		return nil
	}

	if !cfg.IsDigdiratorEnabled() {
		return fmt.Errorf("idporten is not enabled for this cluster")
	}

	// create idporten client and attach secrets
	idportenClient, err := client(resource.CreateObjectMeta(app), idPorten, ingresses)
	if err != nil {
		return err
	}

	ast.Labels["idporten"] = "enabled"
	ast.AppendOperation(resource.OperationCreateOrUpdate, idportenClient)
	pod.WithAdditionalSecret(ast, idportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, idportenClient.Spec.SecretName)

	// return early if sidecar is not enabled
	if idPorten.Sidecar == nil || !idPorten.Sidecar.Enabled {
		return nil
	}

	if !cfg.IsWonderwallEnabled() {
		return fmt.Errorf("idporten sidecar is not enabled for this cluster")
	}

	// create sidecar container
	wonderwallCfg := makeWonderwallConfig(app, idportenClient.Spec.SecretName)
	err = wonderwall.Create(app, ast, cfg, wonderwallCfg)
	if err != nil {
		return err
	}

	// override uris when sidecar is enabled
	idportenClient.Spec.FrontchannelLogoutURI = idportenURI(ingresses, wonderwall.FrontChannelLogoutPath)
	idportenClient.Spec.RedirectURIs = idportenURIs(ingresses, wonderwall.RedirectURIPath)
	idportenClient.Spec.PostLogoutRedirectURIs = idportenURIs(ingresses, wonderwall.LogoutCallbackPath)

	return nil
}

func makeWonderwallConfig(source Source, providerSecretName string) wonderwall.Configuration {
	naisIngresses := source.GetIngress()
	naisIdPorten := source.GetIDPorten()

	cfg := wonderwall.Configuration{
		ACRValues:            naisIdPorten.Sidecar.Level,
		AutoLogin:            naisIdPorten.Sidecar.AutoLogin,
		AutoLoginIgnorePaths: naisIdPorten.Sidecar.AutoLoginIgnorePaths,
		ErrorPath:            naisIdPorten.Sidecar.ErrorPath,
		Ingresses: []string{
			string(naisIngresses[0]),
		},
		Provider:       "idporten",
		SecretNames:    []string{providerSecretName, wonderwallSecretName},
		Resources:      naisIdPorten.Sidecar.Resources,
		SessionRefresh: false,
		UILocales:      naisIdPorten.Sidecar.Locale,
	}

	return cfg
}
