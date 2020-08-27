package resourcecreator

import (
	azureapp "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"path"
)

const (
	AzureApplicationTenantNav = "nav.no"
)

func AzureAdApplication(app nais.Application, options ResourceOptions) azureapp.AzureAdApplication {
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
			PreAuthorizedApplications: accessPolicyRulesWithDefaults(app.Spec.AccessPolicy.Inbound.Rules, app.Namespace, options.ClusterName),
			Tenant:                    getTenant(app),
			SecretName:                getSecretName(app),
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

func oauthCallbackURLs(ingresses []string) []string {
	urls := make([]string, len(ingresses))
	for i := range ingresses {
		u, _ := url.Parse(ingresses[i])
		u.Path = path.Join(u.Path, "/oauth2/callback")
		urls[i] = u.String()
	}
	return urls
}

func getTenant(app nais.Application) string {
	tenant := app.Spec.Azure.Application.Tenant
	if len(tenant) == 0 {
		return AzureApplicationTenantNav
	}
	return tenant
}
