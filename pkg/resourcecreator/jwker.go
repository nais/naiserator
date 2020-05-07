package resourcecreator

import (
	"fmt"
	"math/rand"

	jwker "github.com/nais/naiserator/pkg/apis/nais.io/v1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func parseAccessPolicyRule(rule nais.AccessPolicyRule, namespaceName, clusterName string) nais.AccessPolicyRule {
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

func parseAccessPolicy(policy *nais.AccessPolicy, namespaceName, clusterName string) *nais.AccessPolicy {
	var inbound []nais.AccessPolicyRule
	var outbound []nais.AccessPolicyRule
	for _, rule := range policy.Inbound.Rules {
		inbound = append(inbound, parseAccessPolicyRule(rule, namespaceName, clusterName))
	}
	for _, rule := range policy.Outbound.Rules {
		outbound = append(outbound, parseAccessPolicyRule(rule, namespaceName, clusterName))
	}
	return &nais.AccessPolicy{
		Inbound: &nais.AccessPolicyInbound{
			Rules: inbound,
		},
		Outbound: &nais.AccessPolicyOutbound{
			Rules: outbound,
		},
	}
}

func getSecretName(app *nais.Application) string {
	return fmt.Sprintf("%s-%s", app.Name, randStringBytes(8))
}

func Jwker(app *nais.Application, clusterName string) *jwker.Jwker {
	if len(app.Spec.AccessPolicy.Inbound.Rules) == 0 && len(app.Spec.AccessPolicy.Outbound.Rules) == 0 {
		fmt.Println("No access policies")
		return nil
	}
	return &jwker.Jwker{
		TypeMeta: v1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: jwker.JwkerSpec{
			AccessPolicy: parseAccessPolicy(app.Spec.AccessPolicy, app.Namespace, clusterName),
			SecretName:   getSecretName(app),
		},
	}
}
