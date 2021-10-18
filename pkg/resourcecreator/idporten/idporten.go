package idporten

import (
	"fmt"
	"strings"
	"time"

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

func client(objectMeta metav1.ObjectMeta, naisIdPorten *nais_io_v1.IDPorten, naisIngresses []nais_io_v1.Ingress) (*nais_io_v1.IDPortenClient, error) {
	if err := validateIngresses(naisIngresses); err != nil {
		return nil, err
	}

	if err := validateRedirectURI(naisIdPorten, naisIngresses); err != nil {
		return nil, err
	}

	secretName, err := idPortenSecretName(objectMeta.Name)
	if err != nil {
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
			SecretName:             secretName,
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

func validateRedirectURI(idPorten *nais_io_v1.IDPorten, ingresses []nais_io_v1.Ingress) error {
	ingress := ingresses[0]
	redirectURI := idPorten.RedirectURI

	if len(redirectURI) == 0 {
		return nil
	}

	if !strings.HasPrefix(string(redirectURI), string(ingress)) {
		return fmt.Errorf("redirect URI ('%s') must be a subpath of the ingress ('%s')", redirectURI, ingress)
	}

	if !strings.HasPrefix(string(redirectURI), "https://") {
		return fmt.Errorf("redirect URI must start with https://")
	}
	return nil
}

func redirectURI(idPorten *nais_io_v1.IDPorten, ingresses []nais_io_v1.Ingress) (redirectURI nais_io_v1.IDPortenURI) {
	redirectURI = idPorten.RedirectURI

	if len(idPorten.RedirectURI) == 0 {
		redirectURI = idportenURI(ingresses, clientDefaultCallbackPath)
	}

	if len(idPorten.RedirectPath) > 0 {
		redirectURI = idportenURI(ingresses, idPorten.RedirectPath)
	}

	return
}

func frontchannelLogoutURI(idPorten *nais_io_v1.IDPorten, ingresses []nais_io_v1.Ingress) (frontchannelLogoutURI nais_io_v1.IDPortenURI) {
	frontchannelLogoutURI = idPorten.FrontchannelLogoutURI

	if len(idPorten.FrontchannelLogoutURI) == 0 {
		frontchannelLogoutURI = idportenURI(ingresses, clientDefaultLogoutPath)
	}

	if len(idPorten.FrontchannelLogoutPath) > 0 {
		frontchannelLogoutURI = idportenURI(ingresses, idPorten.FrontchannelLogoutPath)
	}

	return
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

func idPortenSecretName(name string) (string, error) {
	basename := fmt.Sprintf("%s-%s", "idporten", name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d", year, week)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(basename, suffix, maxLen)
}

func Create(app *nais_io_v1alpha1.Application, ast *resource.Ast, resourceOptions resource.Options) error {
	naisIdPorten := app.Spec.IDPorten
	naisIngresses := app.Spec.Ingresses
	if !resourceOptions.DigdiratorEnabled || naisIdPorten == nil || !naisIdPorten.Enabled {
		return nil
	}

	// create idporten client and attach secrets
	idportenClient, err := client(resource.CreateObjectMeta(app), naisIdPorten, naisIngresses)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, idportenClient)

	pod.WithAdditionalSecret(ast, idportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, idportenClient.Spec.SecretName)

	if naisIdPorten.Sidecar == nil || !naisIdPorten.Sidecar.Enabled || !resourceOptions.WonderwallEnabled {
		return nil
	}

	// create sidecar container
	wonderwallCfg := wonderwallConfig(naisIngresses, naisIdPorten, idportenClient.Spec.SecretName)
	err = wonderwall.Create(app, ast, resourceOptions, wonderwallCfg)
	if err != nil {
		return err
	}

	// override uris when sidecar is enabled
	idportenClient.Spec.FrontchannelLogoutURI = idportenURI(naisIngresses, wonderwall.FrontChannelLogoutPath)
	idportenClient.Spec.RedirectURI = idportenURI(naisIngresses, wonderwall.RedirectURIPath)

	return nil
}

func wonderwallConfig(naisIngresses []nais_io_v1.Ingress, naisIdPorten *nais_io_v1.IDPorten, providerSecretName string) wonderwall.Configuration {
	cfg := wonderwall.Configuration{
		AutoLogin:             naisIdPorten.Sidecar.AutoLogin,
		ErrorPath:             naisIdPorten.Sidecar.ErrorPath,
		Ingress:               string(naisIngresses[0]),
		Provider:              "idporten",
		ProviderSecretName:    providerSecretName,
		ACRValues:             naisIdPorten.Sidecar.Level,
		UILocales:             naisIdPorten.Sidecar.Locale,
	}

	if len(naisIdPorten.PostLogoutRedirectURIs) > 0 {
		cfg.PostLogoutRedirectURI = string(naisIdPorten.PostLogoutRedirectURIs[0])
	}

	return cfg
}
