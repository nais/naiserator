package accesspolicy

import "github.com/nais/liberator/pkg/apis/nais.io/v1"

func RulesWithDefaults(rules []nais_io_v1.AccessPolicyRule, namespace, clusterName string) []nais_io_v1.AccessPolicyRule {
	mangled := make([]nais_io_v1.AccessPolicyRule, len(rules))
	for i := range rules {
		mangled[i] = ruleWithDefaults(rules[i], namespace, clusterName)
	}
	return mangled
}

func InboundRulesWithDefaults(rules []nais_io_v1.AccessPolicyInboundRule, namespace, clusterName string) []nais_io_v1.AccessPolicyInboundRule {
	mangled := make([]nais_io_v1.AccessPolicyInboundRule, len(rules))
	for i := range rules {
		rule := rules[i]
		rule.AccessPolicyRule = ruleWithDefaults(rules[i].AccessPolicyRule, namespace, clusterName)
		mangled[i] = rule
	}
	return mangled
}

func ExtractAccessPolicyRules(rules []nais_io_v1.AccessPolicyInboundRule) []nais_io_v1.AccessPolicyRule {
	extracted := make([]nais_io_v1.AccessPolicyRule, len(rules))
	for i := range rules {
		extracted[i] = rules[i].AccessPolicyRule
	}
	return extracted
}

func ruleWithDefaults(rule nais_io_v1.AccessPolicyRule, namespaceName, clusterName string) nais_io_v1.AccessPolicyRule {
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

func WithDefaults(policy *nais_io_v1.AccessPolicy, namespaceName, clusterName string) *nais_io_v1.AccessPolicy {
	return &nais_io_v1.AccessPolicy{
		Inbound: &nais_io_v1.AccessPolicyInbound{
			Rules: InboundRulesWithDefaults(policy.Inbound.Rules, namespaceName, clusterName),
		},
		Outbound: &nais_io_v1.AccessPolicyOutbound{
			Rules: RulesWithDefaults(policy.Outbound.Rules, namespaceName, clusterName),
		},
	}
}
