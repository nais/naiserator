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

func mapAccessPolicy(accessPolicy *nais.AccessPolicy) *jwker.AccessPolicy {
	var jwkerInbound []jwker.AccessPolicyRule
	var jwkerOutbound []jwker.AccessPolicyRule
	for _, inboundRule := range accessPolicy.Inbound.Rules {
		tmp := jwker.AccessPolicyRule{
			Application: inboundRule.Application,
			Namespace:   inboundRule.Namespace,
			Cluster:     inboundRule.Cluster,
		}
		jwkerInbound = append(jwkerInbound, tmp)
	}
	for _, outboundRule := range accessPolicy.Outbound.Rules {
		tmp := jwker.AccessPolicyRule{
			Application: outboundRule.Application,
			Namespace:   outboundRule.Namespace,
			Cluster:     outboundRule.Cluster,
		}
		jwkerOutbound = append(jwkerOutbound, tmp)
	}
	return &jwker.AccessPolicy{
		Inbound: &jwker.AccessPolicyInbound{
			Rules: jwkerInbound,
		},
		Outbound: &jwker.AccessPolicyOutbound{
			Rules: jwkerOutbound,
		},
	}
}

func getSecretName(app *nais.Application) string {
	return fmt.Sprintf("%s-%s", app.Name, randStringBytes(8))
}

func Jwker(app *nais.Application) *jwker.Jwker {
	return &jwker.Jwker{
		TypeMeta: v1.TypeMeta{
			Kind:       "Jwker",
			APIVersion: "jwker.nais.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: jwker.JwkerSpec{
			AccessPolicy: mapAccessPolicy(app.Spec.AccessPolicy),
			SecretName:   getSecretName(app),
		},
	}
}
