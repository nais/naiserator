package resourcecreator

import (
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func accessPolicyRulesWithDefaults(rules []nais_io_v1.AccessPolicyRule, namespace, clusterName string) []nais_io_v1.AccessPolicyRule {
	mangled := make([]nais_io_v1.AccessPolicyRule, len(rules))
	for i := range rules {
		mangled[i] = accessPolicyRuleWithDefaults(rules[i], namespace, clusterName)
	}
	return mangled
}

func accessPolicyRuleWithDefaults(rule nais_io_v1.AccessPolicyRule, namespaceName, clusterName string) nais_io_v1.AccessPolicyRule {
	accessPolicyRule := nais_io_v1.AccessPolicyRule{
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

func accessPoliciesWithDefaults(policy *nais_io_v1.AccessPolicy, namespaceName, clusterName string) *nais_io_v1.AccessPolicy {
	return &nais_io_v1.AccessPolicy{
		Inbound: &nais_io_v1.AccessPolicyInbound{
			Rules: accessPolicyRulesWithDefaults(policy.Inbound.Rules, namespaceName, clusterName),
		},
		Outbound: &nais_io_v1.AccessPolicyOutbound{
			Rules: accessPolicyRulesWithDefaults(policy.Outbound.Rules, namespaceName, clusterName),
		},
	}
}

func jwkerSecretName(app nais_io_v1alpha1.Application) string {
	return namegen.PrefixedRandShortName("tokenx", app.Name, validation.DNS1035LabelMaxLength)
}

func Jwker(app *nais_io_v1alpha1.Application, clusterName string) *nais_io_v1.Jwker {
	return &nais_io_v1.Jwker{
		TypeMeta: v1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: nais_io_v1.JwkerSpec{
			AccessPolicy: accessPoliciesWithDefaults(app.Spec.AccessPolicy, app.Namespace, clusterName),
			SecretName:   jwkerSecretName(*app),
		},
	}
}
