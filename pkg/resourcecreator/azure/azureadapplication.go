package azure

import (
	"fmt"
	"strings"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/pointer"

	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/wonderwall"
	"github.com/nais/naiserator/pkg/util"
)

const (
	applicationDefaultCallbackPath = "/oauth2/callback"
)

type Source interface {
	resource.Source
	wonderwall.Source
	GetAccessPolicy() *nais_io_v1.AccessPolicy
	GetAzure() nais_io_v1.AzureInterface
	GetIngress() []nais_io_v1.Ingress
}

type Config interface {
	wonderwall.Config
	GetClusterName() string
	IsAzureratorEnabled() bool
	IsWonderwallEnabled() bool
}

func Create(source Source, ast *resource.Ast, config Config) error {
	if shouldNotCreate(source, config) {
		return nil
	}

	azureAdApplication, err := application(source, config)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, azureAdApplication)
	pod.WithAdditionalSecret(ast, azureAdApplication.Spec.SecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
	pod.WithAdditionalEnvFromSecret(ast, azureAdApplication.Spec.SecretName)

	if shouldNotCreateWonderwall(source, config) {
		return nil
	}

	// configure sidecar
	ingresses := source.GetIngress()
	if len(ingresses) == 0 {
		return fmt.Errorf("must have at least 1 ingress to use Azure AD sidecar")
	}

	wwCfg := wonderwallConfig(source, azureAdApplication.Spec.SecretName, ingresses)
	err = wonderwall.Create(source, ast, config, wwCfg)
	if err != nil {
		return err
	}

	// ensure that the ingress is added to the configured Azure AD reply URLs
	azureAdApplication.Spec.ReplyUrls = append(azureAdApplication.Spec.ReplyUrls, mapReplyURLs(callbackURLs(ingresses))...)
	azureAdApplication.Spec.LogoutUrl = util.AppendPathToIngress(ingresses[0], wonderwall.FrontChannelLogoutPath)

	// ensure that singlePageApplication is _disabled_ if sidecar is enabled
	azureAdApplication.Spec.SinglePageApplication = pointer.Bool(false)

	return nil
}

func shouldNotCreate(source Source, config Config) bool {
	az := source.GetAzure()
	return !config.IsAzureratorEnabled() || az.GetApplication() == nil || !az.GetApplication().Enabled
}

func shouldNotCreateWonderwall(source Source, config Config) bool {
	az := source.GetAzure()
	return !config.IsWonderwallEnabled() || az.GetSidecar() == nil || !az.GetSidecar().Enabled
}

func application(source Source, config Config) (*nais_io_v1.AzureAdApplication, error) {
	replyURLs := source.GetAzure().GetApplication().ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = callbackURLs(source.GetIngress())
	}

	secretName, err := secretName(source.GetName())
	if err != nil {
		return &nais_io_v1.AzureAdApplication{}, err
	}

	objectMeta := resource.CreateObjectMeta(source)
	copyAzureAnnotations(source.GetAnnotations(), objectMeta.Annotations)

	preAuthorizedApps := accesspolicy.InboundRulesWithDefaults(source.GetAccessPolicy().Inbound.Rules, objectMeta.Namespace, config.GetClusterName())

	azureapp := source.GetAzure().GetApplication()

	return &nais_io_v1.AzureAdApplication{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: preAuthorizedApps,
			Tenant:                    azureapp.Tenant,
			SecretName:                secretName,
			Claims:                    azureapp.Claims,
			SinglePageApplication:     azureapp.SinglePageApplication,
			AllowAllUsers:             azureapp.AllowAllUsers,
		},
	}, nil
}

func copyAzureAnnotations(src, dst map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "azure.nais.io/") {
			dst[k] = v
		}
	}
}

func mapReplyURLs(urls []nais_io_v1.AzureAdReplyUrlString) []nais_io_v1.AzureAdReplyUrl {
	maps := make([]nais_io_v1.AzureAdReplyUrl, len(urls))
	for i := range urls {
		maps[i].Url = urls[i]
	}
	return maps
}

func callbackURLs(ingresses []nais_io_v1.Ingress) []nais_io_v1.AzureAdReplyUrlString {
	urls := make([]nais_io_v1.AzureAdReplyUrlString, len(ingresses))
	for i := range ingresses {
		urls[i] = appendPathToIngress(ingresses[i], applicationDefaultCallbackPath)
	}
	return urls
}

func secretName(name string) (string, error) {
	prefixedName := fmt.Sprintf("%s-%s", "azure", name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d", year, week)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func appendPathToIngress(url nais_io_v1.Ingress, path string) nais_io_v1.AzureAdReplyUrlString {
	return (nais_io_v1.AzureAdReplyUrlString)(util.AppendPathToIngress(url, path))
}

func wonderwallConfig(source Source, providerSecretName string, ingresses []nais_io_v1.Ingress) wonderwall.Configuration {
	sidecar := source.GetAzure().GetSidecar()

	ingressesStrings := make([]string, 0)
	for _, i := range ingresses {
		ingressesStrings = append(ingressesStrings, string(i))
	}

	return wonderwall.Configuration{
		AutoLogin:          sidecar.AutoLogin,
		ErrorPath:          sidecar.ErrorPath,
		Ingresses:          ingressesStrings,
		Loginstatus:        false,
		Provider:           "azure",
		ProviderSecretName: providerSecretName,
		Resources:          sidecar.Resources,
		SessionRefresh:     true,
	}
}
