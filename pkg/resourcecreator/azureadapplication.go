package resourcecreator

import (
	"net/url"
	"path"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AzureAdApplication(app nais.Application, options ResourceOptions) nais.AzureAdApplication {
	replyURLs := app.Spec.Azure.Application.ReplyURLs
	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(app.Spec.Ingresses)
	}
	return nais.AzureAdApplication{
		TypeMeta: v1.TypeMeta{
			Kind:       "AzureAdApplication",
			APIVersion: "nais.io/v1alpha1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: nais.AzureAdApplicationSpec{
			ReplyUrls:                 mapReplyURLs(replyURLs),
			PreAuthorizedApplications: accessPolicyRulesWithDefaults(app.Spec.AccessPolicy.Inbound.Rules, app.Namespace, options.ClusterName),
			LogoutUrl:                 app.Spec.Azure.Application.LogoutURL,
			SecretName:                getSecretName(app),
		},
	}
}

func mapReplyURLs(urls []string) []nais.AzureAdReplyUrl {
	maps := make([]nais.AzureAdReplyUrl, len(urls))
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
