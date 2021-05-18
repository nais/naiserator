package idporten

import (
	"fmt"
	"strings"

	idportenClient "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	clientDefaultCallbackPath = "/oauth2/callback"
	clientDefaultLogoutPath   = "/oauth2/logout"
)

func client(app *nais.Application) (*idportenClient.IDPortenClient, error) {
	if err := validateIngresses(app); err != nil {
		return nil, err
	}

	if err := validateRedirectURI(app); err != nil {
		return nil, err
	}

	return &idportenClient.IDPortenClient{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IDPortenClient",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: idportenClient.IDPortenClientSpec{
			ClientURI:              app.Spec.IDPorten.ClientURI,
			RedirectURI:            redirectURI(app),
			SecretName:             idPortenSecretName(*app),
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
		redirectURI = util.AppendPathToIngress(app.Spec.Ingresses[0], clientDefaultCallbackPath)
	}

	if len(app.Spec.IDPorten.RedirectPath) > 0 {
		redirectURI = util.AppendPathToIngress(app.Spec.Ingresses[0], app.Spec.IDPorten.RedirectPath)
	}

	return
}

func frontchannelLogoutURI(app *nais.Application) (frontchannelLogoutURI string) {
	frontchannelLogoutURI = app.Spec.IDPorten.FrontchannelLogoutURI

	if len(app.Spec.IDPorten.FrontchannelLogoutURI) == 0 {
		frontchannelLogoutURI = util.AppendPathToIngress(app.Spec.Ingresses[0], clientDefaultLogoutPath)
	}

	if len(app.Spec.IDPorten.FrontchannelLogoutPath) > 0 {
		frontchannelLogoutURI = util.AppendPathToIngress(app.Spec.Ingresses[0], app.Spec.IDPorten.FrontchannelLogoutPath)
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

func idPortenSecretName(app nais.Application) string {
	return namegen.PrefixedRandShortName("idporten", app.Name, validation.DNS1035LabelMaxLength)
}

func Create(app *nais.Application, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) error {
	if resourceOptions.DigdiratorEnabled && app.Spec.IDPorten != nil && app.Spec.IDPorten.Enabled {
		idportenClient, err := client(app)
		if err != nil {
			return err
		}

		*operations = append(*operations, resource.Operation{Resource: idportenClient, Operation: resource.OperationCreateOrUpdate})

		podSpec := &deployment.Spec.Template.Spec
		podSpec = pod.WithAdditionalSecret(podSpec, idportenClient.Spec.SecretName, nais.DefaultDigdiratorIDPortenMountPath)
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, idportenClient.Spec.SecretName)
		deployment.Spec.Template.Spec = *podSpec
	}

	return nil
}
