package idporten

import (
	"fmt"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
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

	return &nais_io_v1.IDPortenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.IDPortenClientSpec{
			ClientURI:              naisIdPorten.ClientURI,
			RedirectURI:            redirectURI(naisIdPorten, naisIngresses),
			SecretName:             idPortenSecretName(objectMeta.Name),
			FrontchannelLogoutURI:  frontchannelLogoutURI(naisIdPorten, naisIngresses),
			PostLogoutRedirectURIs: postLogoutRedirectURIs(naisIdPorten),
			SessionLifetime:        naisIdPorten.SessionLifetime,
			AccessTokenLifetime:    naisIdPorten.AccessTokenLifetime,
		},
	}, nil
}

func validateIngresses(ingresser []nais_io_v1.Ingress) error {
	if len(ingresser) == 0 {
		return fmt.Errorf("you must specify an ingress to be able to use the idporten integration")
	}

	if len(ingresser) > 1 {
		return fmt.Errorf("cannot have more than one ingress when using the idporten integration")
	}
	return nil
}

func validateRedirectURI(idPorten *nais_io_v1.IDPorten, ingresser []nais_io_v1.Ingress) error {
	ingress := ingresser[0]
	redirectURI := idPorten.RedirectURI

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

func redirectURI(idPorten *nais_io_v1.IDPorten, ingresser []nais_io_v1.Ingress) (redirectURI string) {
	redirectURI = idPorten.RedirectURI

	if len(idPorten.RedirectURI) == 0 {
		redirectURI = util.AppendPathToIngress(ingresser[0], clientDefaultCallbackPath)
	}

	if len(idPorten.RedirectPath) > 0 {
		redirectURI = util.AppendPathToIngress(ingresser[0], idPorten.RedirectPath)
	}

	return
}

func frontchannelLogoutURI(idPorten *nais_io_v1.IDPorten, ingresser []nais_io_v1.Ingress) (frontchannelLogoutURI string) {
	frontchannelLogoutURI = idPorten.FrontchannelLogoutURI

	if len(idPorten.FrontchannelLogoutURI) == 0 {
		frontchannelLogoutURI = util.AppendPathToIngress(ingresser[0], clientDefaultLogoutPath)
	}

	if len(idPorten.FrontchannelLogoutPath) > 0 {
		frontchannelLogoutURI = util.AppendPathToIngress(ingresser[0], idPorten.FrontchannelLogoutPath)
	}

	return
}

func postLogoutRedirectURIs(idPorten *nais_io_v1.IDPorten) (postLogoutRedirectURIs []string) {
	postLogoutRedirectURIs = idPorten.PostLogoutRedirectURIs

	if len(idPorten.PostLogoutRedirectURIs) == 0 {
		postLogoutRedirectURIs = []string{}
	}

	return
}

func idPortenSecretName(name string) string {
	return namegen.PrefixedRandShortName("idporten", name, validation.DNS1035LabelMaxLength)
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisIdPorten *nais_io_v1.IDPorten, naisIngresses []nais_io_v1.Ingress) error {
	if resourceOptions.DigdiratorEnabled && naisIdPorten != nil && naisIdPorten.Enabled {

		idportenClient, err := client(resource.CreateObjectMeta(source), naisIdPorten, naisIngresses)
		if err != nil {
			return err
		}

		ast.AppendOperation(resource.OperationCreateOrUpdate, idportenClient)

		pod.WithAdditionalSecret(ast, idportenClient.Spec.SecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
		pod.WithAdditionalEnvFromSecret(ast, idportenClient.Spec.SecretName)
	}

	return nil
}
