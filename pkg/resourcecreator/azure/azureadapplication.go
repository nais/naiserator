package azure

import (
	"fmt"
	"time"

	azureapp "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	applicationDefaultCallbackPath = "/oauth2/callback"
)

func adApplication(objectMeta metav1.ObjectMeta, naisAzure nais_io_v1.Azure, naisIngress []nais_io_v1.Ingress, naisAccessPolicy nais_io_v1.AccessPolicy, clusterName string) (*azureapp.AzureAdApplication, error) {
	replyURLs := naisAzure.Application.ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(naisIngress)
	}

	secretName, err := azureSecretName(objectMeta.Name)
	if err != nil {
		return &azureapp.AzureAdApplication{}, err
	}

	return &azureapp.AzureAdApplication{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: azureapp.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: accesspolicy.RulesWithDefaults(naisAccessPolicy.Inbound.Rules, objectMeta.Namespace, clusterName),
			Tenant:                    naisAzure.Application.Tenant,
			SecretName:                secretName,
			Claims:                    naisAzure.Application.Claims,
		},
	}, nil
}

func mapReplyURLs(urls []string) []azureapp.AzureAdReplyUrl {
	maps := make([]azureapp.AzureAdReplyUrl, len(urls))
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
	maxLen -= len(suffix) + 1 // length of suffix + 1 byte of separator

	shortName, err := namegen.ShortName(prefixedName, maxLen)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", shortName, suffix), nil
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisAzure nais_io_v1.Azure, naisIngress []nais_io_v1.Ingress, naisAccessPolicy nais_io_v1.AccessPolicy) error {
	if !resourceOptions.AzureratorEnabled || !naisAzure.Application.Enabled {
		return nil
	}

	azureAdApplication, err := adApplication(resource.CreateObjectMeta(source), naisAzure, naisIngress, naisAccessPolicy, resourceOptions.ClusterName)
	if err != nil {
		return err
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, azureAdApplication)

	pod.WithAdditionalSecret(ast, azureAdApplication.Spec.SecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
	pod.WithAdditionalEnvFromSecret(ast, azureAdApplication.Spec.SecretName)

	return nil
}
