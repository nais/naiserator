package azure

import (
	"fmt"
	"time"

	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"

	azureapp "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	applicationDefaultCallbackPath = "/oauth2/callback"
)

func adApplication(app nais_io_v1alpha1.Application, clusterName string) (*azureapp.AzureAdApplication, error) {
	replyURLs := app.Spec.Azure.Application.ReplyURLs

	if len(replyURLs) == 0 {
		replyURLs = oauthCallbackURLs(app.Spec.Ingresses)
	}

	secretName, err := azureSecretName(app)
	if err != nil {
		return &azureapp.AzureAdApplication{}, err
	}

	return &azureapp.AzureAdApplication{
		TypeMeta: metav1.TypeMeta{
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

func oauthCallbackURLs(ingresses []nais_io_v1alpha1.Ingress) []string {
	urls := make([]string, len(ingresses))
	for i := range ingresses {
		urls[i] = util.AppendPathToIngress(ingresses[i], applicationDefaultCallbackPath)
	}
	return urls
}

func azureSecretName(app nais_io_v1alpha1.Application) (string, error) {
	prefixedName := fmt.Sprintf("%s-%s", "azure", app.Name)
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

func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) error {
	if resourceOptions.AzureratorEnabled && app.Spec.Azure.Application.Enabled {
		azureAdApplication, err := adApplication(*app, resourceOptions.ClusterName)
		if err != nil {
			return err
		}

		*operations = append(*operations, resource.Operation{Resource: azureAdApplication, Operation: resource.OperationCreateOrUpdate})

		podSpec := &deployment.Spec.Template.Spec
		podSpec = pod.WithAdditionalSecret(podSpec, azureAdApplication.Spec.SecretName, nais_io_v1alpha1.DefaultAzureratorMountPath)
		podSpec = pod.WithAdditionalEnvFromSecret(podSpec, azureAdApplication.Spec.SecretName)
		deployment.Spec.Template.Spec = *podSpec
	}

	return nil
}
