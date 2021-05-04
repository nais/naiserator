package resourcecreator

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
)

func ToAccessPolicyExternalRules(hosts []string) []nais_io_v1.AccessPolicyExternalRule {
	rules := make([]nais_io_v1.AccessPolicyExternalRule, 0)

	for _, host := range hosts {
		rules = append(rules, nais_io_v1.AccessPolicyExternalRule{
			Host: host,
		})
	}
	return rules
}

func MergeExternalRules(app *nais_io_v1alpha1.Application, additionalRules ...nais_io_v1.AccessPolicyExternalRule) []nais_io_v1.AccessPolicyExternalRule {
	rules := app.Spec.AccessPolicy.Outbound.External
	if len(additionalRules) == 0 {
		return rules
	}

	var empty struct{}
	seen := map[string]struct{}{}

	for _, externalRule := range rules {
		seen[externalRule.Host] = empty
	}

	for _, rule := range additionalRules {
		if len(rule.Host) == 0 {
			continue
		}

		if _, found := seen[rule.Host]; !found {
			seen[rule.Host] = empty
			rules = append(rules, rule)
		}
	}

	return rules
}
