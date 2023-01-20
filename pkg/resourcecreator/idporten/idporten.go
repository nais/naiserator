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

func Create(source Source, ast *resource.Ast, cfg Config) error {
	idporten := source.GetIDPorten()
	if idporten == nil {
		return nil
	}

	sidecarEnabled := idporten.Sidecar != nil && idporten.Sidecar.Enabled
	if !idporten.Enabled && !sidecarEnabled {
		return nil
	}

	if !cfg.IsDigdiratorEnabled() {
		return fmt.Errorf("idporten is not available in this cluster")
	}

	// create idporten client and attach secrets
	idportenClient, err := client(source)
	if err != nil {
		return err
	}

	ast.Labels["idporten"] = "enabled"
	ast.AppendOperation(resource.OperationCreateOrUpdate, idportenClient)
	pod.WithAdditionalSecret(ast, idportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, idportenClient.Spec.SecretName)

	if sidecarEnabled {
		return sidecar(source, ast, cfg, idportenClient)
	}

	return nil
}

func client(source Source) (*nais_io_v1.IDPortenClient, error) {
	objectMeta := resource.CreateObjectMeta(source)
	idporten := source.GetIDPorten()
	ingresses := source.GetIngress()

	if err := validateIngresses(ingresses); err != nil {
		return nil, err
	}

	name, err := secretName(objectMeta.Name)
	if err != nil {
		return nil, fmt.Errorf("generate secret name: %w", err)
	}

	return &nais_io_v1.IDPortenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.IDPortenClientSpec{
			ClientURI:              idporten.ClientURI,
			IntegrationType:        idporten.IntegrationType,
			RedirectURIs:           redirectURIs(idporten, ingresses),
			SecretName:             name,
			FrontchannelLogoutURI:  frontchannelLogoutURI(idporten, ingresses),
			PostLogoutRedirectURIs: postLogoutRedirectURIs(idporten),
			SessionLifetime:        idporten.SessionLifetime,
			AccessTokenLifetime:    idporten.AccessTokenLifetime,
			Scopes:                 idporten.Scopes,
		},
	}, nil
}

func sidecar(source Source, ast *resource.Ast, cfg Config, idportenClient *nais_io_v1.IDPortenClient) error {
	if !cfg.IsWonderwallEnabled() {
		return fmt.Errorf("idporten sidecar is not enabled for this cluster")
	}

	// create sidecar container
	wonderwallCfg := makeWonderwallConfig(source, idportenClient.Spec.SecretName)
	err := wonderwall.Create(source, ast, cfg, wonderwallCfg)
	if err != nil {
		return err
	}

	// override uris when sidecar is enabled
	ingresses := source.GetIngress()
	idportenClient.Spec.FrontchannelLogoutURI = idportenURI(ingresses, wonderwall.FrontChannelLogoutPath)
	idportenClient.Spec.RedirectURIs = idportenURIs(ingresses, wonderwall.RedirectURIPath)
	idportenClient.Spec.PostLogoutRedirectURIs = idportenURIs(ingresses, wonderwall.LogoutCallbackPath)

	return nil
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

func secretName(name string) (string, error) {
	return namegen.ShortName(fmt.Sprintf("idporten-%s", name), validation.DNS1035LabelMaxLength)
}

func makeWonderwallConfig(source Source, providerSecretName string) wonderwall.Configuration {
	ingresses := source.GetIngress()
	idporten := source.GetIDPorten()

	ingressesStrings := make([]string, 0)
	for _, i := range ingresses {
		ingressesStrings = append(ingressesStrings, string(i))
	}

	cfg := wonderwall.Configuration{
		ACRValues:            idporten.Sidecar.Level,
		AutoLogin:            idporten.Sidecar.AutoLogin,
		AutoLoginIgnorePaths: idporten.Sidecar.AutoLoginIgnorePaths,
		ErrorPath:            idporten.Sidecar.ErrorPath,
		Ingresses:            ingressesStrings,
		Provider:             "idporten",
		SecretNames:          []string{providerSecretName, wonderwallSecretName},
		Resources:            idporten.Sidecar.Resources,
		SessionRefresh:       false,
		UILocales:            idporten.Sidecar.Locale,
	}

	return cfg
}
