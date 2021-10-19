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
	GetAccessPolicy() *nais_io_v1.AccessPolicy
	GetAzure() nais_io_v1.AzureInterface
	GetPort() int
	GetIngress() []nais_io_v1.Ingress
}

func adApplication(source Source, resourceOptions resource.Options) (*nais_io_v1.AzureAdApplication, error) {
	replyURLs := source.GetAzure().GetApplication().ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(source.GetIngress())
	}

	secretName, err := azureSecretName(source.GetName())
	if err != nil {
		return &nais_io_v1.AzureAdApplication{}, err
	}

	objectMeta := resource.CreateObjectMeta(source)
	copyAzureAnnotations(source.GetAnnotations(), objectMeta.Annotations)

	clusterName := resourceOptions.ClusterName
	preAuthorizedApps := accesspolicy.InboundRulesWithDefaults(source.GetAccessPolicy().Inbound.Rules, objectMeta.Namespace, clusterName)

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

func mapReplyURLs(urls []string) []nais_io_v1.AzureAdReplyUrl {
	maps := make([]nais_io_v1.AzureAdReplyUrl, len(urls))
	for i := range urls {
		maps[i].Url = urls[i]
	}
	return maps
}

func oauthCallbackURLs(ingresses []nais_io_v1.Ingress) []string {
	urls := make([]string, len(ingresses))
	for i := range ingresses {
		urls[i] = util.AppendPathToIngress(ingresses[i], applicationDefaultCallbackPath)
	}
	return urls
}

func azureSecretName(name string) (string, error) {
	prefixedName := fmt.Sprintf("%s-%s", "azure", name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d", year, week)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func Create(source Source, ast *resource.Ast, resourceOptions resource.Options) error {
	az := source.GetAzure()

	if !resourceOptions.AzureratorEnabled || az.GetApplication() == nil || !az.GetApplication().Enabled {
		return nil
	}

	azureAdApplication, err := adApplication(source, resourceOptions)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, azureAdApplication)

	pod.WithAdditionalSecret(ast, azureAdApplication.Spec.SecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
	pod.WithAdditionalEnvFromSecret(ast, azureAdApplication.Spec.SecretName)

	if !resourceOptions.WonderwallEnabled || az.GetSidecar() == nil || !az.GetSidecar().Enabled {
		return nil
	}

	// configure sidecar
	ingresses := source.GetIngress()
	if len(ingresses) == 0 {
		return fmt.Errorf("must have at least 1 ingress to use Azure AD sidecar")
	}

	// wonderwall only supports a single ingress, so we use the first
	ingress := ingresses[0]

	config := wonderwallConfig(source, azureAdApplication.Spec.SecretName, ingress)
	err = wonderwall.Create(source, ast, resourceOptions, config)
	if err != nil {
		return err
	}

	// ensure that the ingress is added to the configured Azure AD reply URLs
	azureAdApplication.Spec.ReplyUrls = append(azureAdApplication.Spec.ReplyUrls, nais_io_v1.AzureAdReplyUrl{
		Url: util.AppendPathToIngress(ingress, applicationDefaultCallbackPath),
	})
	azureAdApplication.Spec.LogoutUrl = util.AppendPathToIngress(ingress, wonderwall.FrontChannelLogoutPath)

	return nil
}

func wonderwallConfig(source Source, providerSecretName string, ingress nais_io_v1.Ingress) wonderwall.Configuration {
	sidecar := source.GetAzure().GetSidecar()
	ing := string(ingress)

	return wonderwall.Configuration{
		AutoLogin:             sidecar.AutoLogin,
		ErrorPath:             sidecar.ErrorPath,
		Ingress:               ing,
		Provider:              "azure",
		ProviderSecretName:    providerSecretName,
	}
}
