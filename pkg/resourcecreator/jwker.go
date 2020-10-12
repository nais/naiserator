package resourcecreator

import (
	"fmt"
	jwker "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func accessPolicyRulesWithDefaults(rules []nais.AccessPolicyRule, namespace, clusterName string) []nais.AccessPolicyRule {
	mangled := make([]nais.AccessPolicyRule, len(rules))
	for i := range rules {
		mangled[i] = accessPolicyRuleWithDefaults(rules[i], namespace, clusterName)
	}
	return mangled
}

func accessPolicyRuleWithDefaults(rule nais.AccessPolicyRule, namespaceName, clusterName string) nais.AccessPolicyRule {
	accessPolicyRule := nais.AccessPolicyRule{
		Application: rule.Application,
		Namespace:   rule.Namespace,
		Cluster:     rule.Cluster,
	}
	if rule.Cluster == "" {
		accessPolicyRule.Cluster = clusterName
	}
	if rule.Namespace == "" {
		accessPolicyRule.Namespace = namespaceName
	}
	return accessPolicyRule
}

func accessPoliciesWithDefaults(policy *nais.AccessPolicy, namespaceName, clusterName string) *nais.AccessPolicy {
	return &nais.AccessPolicy{
		Inbound: &nais.AccessPolicyInbound{
			Rules: accessPolicyRulesWithDefaults(policy.Inbound.Rules, namespaceName, clusterName),
		},
		Outbound: &nais.AccessPolicyOutbound{
			Rules: accessPolicyRulesWithDefaults(policy.Outbound.Rules, namespaceName, clusterName),
		},
	}
}

func getSecretName(app nais.Application) string {
	return fmt.Sprintf("%s-%s", app.Name, util.RandStringBytes(8))
}

func Jwker(app *nais.Application, clusterName string) *jwker.Jwker {
	return &jwker.Jwker{
		TypeMeta: v1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: jwker.JwkerSpec{
			AccessPolicy: accessPoliciesWithDefaults(app.Spec.AccessPolicy, app.Namespace, clusterName),
			SecretName:   getSecretName(*app),
		},
	}
}
