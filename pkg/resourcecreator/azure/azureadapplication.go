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
	"github.com/nais/naiserator/pkg/util"
)

const (
	applicationDefaultCallbackPath = "/oauth2/callback"
)

func adApplication(objectMeta metav1.ObjectMeta, naisAzure nais_io_v1.Azure, naisIngress []nais_io_v1.Ingress, naisAccessPolicy nais_io_v1.AccessPolicy, clusterName string, annotations map[string]string) (*nais_io_v1.AzureAdApplication, error) {
	replyURLs := naisAzure.Application.ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(naisIngress)
	}

	secretName, err := azureSecretName(objectMeta.Name)
	if err != nil {
		return &nais_io_v1.AzureAdApplication{}, err
	}

	copyAzureAnnotations(annotations, objectMeta.Annotations)

	return &nais_io_v1.AzureAdApplication{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: nais_io_v1.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: accesspolicy.InboundRulesWithDefaults(naisAccessPolicy.Inbound.Rules, objectMeta.Namespace, clusterName),
			Tenant:                    naisAzure.Application.Tenant,
			SecretName:                secretName,
			Claims:                    naisAzure.Application.Claims,
			SinglePageApplication:     naisAzure.Application.SinglePageApplication,
			AllowAllUsers:             naisAzure.Application.AllowAllUsers,
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

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisAzure nais_io_v1.Azure, naisIngress []nais_io_v1.Ingress, naisAccessPolicy nais_io_v1.AccessPolicy) error {
	if !resourceOptions.AzureratorEnabled || !naisAzure.Application.Enabled {
		return nil
	}

	azureAdApplication, err := adApplication(resource.CreateObjectMeta(source), naisAzure, naisIngress, naisAccessPolicy, resourceOptions.ClusterName, source.GetAnnotations())
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, azureAdApplication)

	pod.WithAdditionalSecret(ast, azureAdApplication.Spec.SecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
	pod.WithAdditionalEnvFromSecret(ast, azureAdApplication.Spec.SecretName)

	return nil
}
