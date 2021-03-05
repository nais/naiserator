package resourcecreator

import (
	"net/url"
	"path"

	azureapp "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AzureApplicationDefaultCallbackPath = "/oauth2/callback"
)

func AzureAdApplication(app nais.Application, clusterName string) (*azureapp.AzureAdApplication, error) {
	replyURLs := app.Spec.Azure.Application.ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(app.Spec.Ingresses)
	}

	secretName, err := util.GenerateSecretName("azure-ad", app.Name, MaxSecretNameLength)
	if err != nil {
		return nil, err
	}

	return &azureapp.AzureAdApplication{
		TypeMeta: v1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: azureapp.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: accessPolicyRulesWithDefaults(app.Spec.AccessPolicy.Inbound.Rules, app.Namespace, clusterName),
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
		urls[i] = appendPathToIngress(ingresses[i], AzureApplicationDefaultCallbackPath)
	}
	return urls
}

func appendPathToIngress(ingress nais.Ingress, joinPath string) string {
	u, _ := url.Parse(string(ingress))
	u.Path = path.Join(u.Path, joinPath)
	return u.String()
}
