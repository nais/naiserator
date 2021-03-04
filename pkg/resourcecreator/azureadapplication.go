package resourcecreator

import (
	"net/url"
	"path"

	azureapp "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AzureApplicationTenantNav           = "nav.no"
	AzureApplicationDefaultCallbackPath = "/oauth2/callback"
)

func AzureAdApplication(app nais.Application, clusterName string) azureapp.AzureAdApplication {
	replyURLs := app.Spec.Azure.Application.ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(app.Spec.Ingresses)
	}

	return azureapp.AzureAdApplication{
		TypeMeta: v1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: azureapp.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: accessPolicyRulesWithDefaults(app.Spec.AccessPolicy.Inbound.Rules, app.Namespace, clusterName),
			Tenant:                    getTenant(app),
			SecretName:                getSecretName(app),
			Claims:                    app.Spec.Azure.Application.Claims,
		},
	}
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

func getTenant(app nais.Application) string {
	tenant := app.Spec.Azure.Application.Tenant
	if len(tenant) == 0 {
		return AzureApplicationTenantNav
	}
	return tenant
}
