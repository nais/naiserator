package resourcecreator

import (
	"fmt"
	"time"

	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"

	azureapp "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	AzureApplicationDefaultCallbackPath = "/oauth2/callback"
)

func AzureAdApplication(app nais.Application, clusterName string) (azureapp.AzureAdApplication, error) {
	replyURLs := app.Spec.Azure.Application.ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(app.Spec.Ingresses)
	}

	secretName, err := azureSecretName(app)
	if err != nil {
		return azureapp.AzureAdApplication{}, err
	}

	return azureapp.AzureAdApplication{
		TypeMeta: v1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: azureapp.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: accesspolicy.RulesWithDefaults(app.Spec.AccessPolicy.Inbound.Rules, app.Namespace, clusterName),
			Tenant:                    app.Spec.Azure.Application.Tenant,
			SecretName:                secretName,
			Claims:                    app.Spec.Azure.Application.Claims,
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

func oauthCallbackURLs(ingresses []nais.Ingress) []string {
	urls := make([]string, len(ingresses))
	for i := range ingresses {
		urls[i] = util.AppendPathToIngress(ingresses[i], AzureApplicationDefaultCallbackPath)
	}
	return urls
}

func azureSecretName(app nais.Application) (string, error) {
	prefixedName := fmt.Sprintf("%s-%s", "azure", app.Name)
	suffix := time.Now().Format("2006-01-02") // YYYY-MM-DD / ISO 8601

	maxLen := validation.DNS1035LabelMaxLength
	maxLen -= len(suffix) + 1 // length of suffix + 1 byte of separator

	shortName, err := namegen.ShortName(prefixedName, maxLen)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", shortName, suffix), nil
}
