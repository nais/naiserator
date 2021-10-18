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
	GetAccessPolicy() nais_io_v1.AccessPolicy
	GetAzure() nais_io_v1.Azure
	GetPort() int
	GetIngress() []nais_io_v1.Ingress
}

func adApplication(source Source, resourceOptions resource.Options) (*nais_io_v1.AzureAdApplication, error) {
	replyURLs := source.GetAzure().Application.ReplyURLs

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

	return &nais_io_v1.AzureAdApplication{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: preAuthorizedApps,
			Tenant:                    source.GetAzure().Application.Tenant,
			SecretName:                secretName,
			Claims:                    source.GetAzure().Application.Claims,
			SinglePageApplication:     source.GetAzure().Application.SinglePageApplication,
			AllowAllUsers:             source.GetAzure().Application.AllowAllUsers,
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
	if !resourceOptions.AzureratorEnabled || !source.GetAzure().Application.Enabled {
		return nil
	}

	azureAdApplication, err := adApplication(source, resourceOptions)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, azureAdApplication)

	pod.WithAdditionalSecret(ast, azureAdApplication.Spec.SecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
	pod.WithAdditionalEnvFromSecret(ast, azureAdApplication.Spec.SecretName)

	if !resourceOptions.WonderwallEnabled || source.GetAzure().Sidecar == nil || !source.GetAzure().Sidecar.Enabled {
		return nil
	}

	config := wonderwallConfig(source, azureAdApplication.Spec.SecretName)
	err = wonderwall.Create(source, ast, resourceOptions, config)
	if err != nil {
		return err
	}

	return nil
}

func wonderwallConfig(source Source, providerSecretName string) wonderwall.Configuration {
	ingress := string(source.GetIngress()[0])

	return wonderwall.Configuration{
		AutoLogin:             source.GetAzure().Sidecar.AutoLogin,
		ErrorPath:             source.GetAzure().Sidecar.ErrorPath,
		Ingress:               ingress,
		Provider:              "azure",
		ProviderSecretName:    providerSecretName,
		PostLogoutRedirectURI: ingress,
	}
}
