package idporten

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
)

const redisImage = "redis:6"
const redisPort = 6379

func Redis(source resource.Source) *nais_io_v1alpha1.Application {
	objectMeta := source.GetObjectMeta()
	objectMeta.SetOwnerReferences(nil)

	return &nais_io_v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "nais.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nais-io-wonderwall-redis",
			Namespace: source.GetNamespace(),
			Labels: map[string]string{
				"team": source.GetLabels()["team"],
			},
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Image: redisImage,
			Port:  redisPort,
			Service: &nais_io_v1.Service{
				Port:     redisPort,
				Protocol: "redis",
			},
			Replicas: &nais_io_v1.Replicas{
				Min: util.Intp(1),
				Max: util.Intp(1),
			},
			AccessPolicy: &nais_io_v1.AccessPolicy{
				Inbound: &nais_io_v1.AccessPolicyInbound{
					Rules: []nais_io_v1.AccessPolicyInboundRule{
						{
							AccessPolicyRule: nais_io_v1.AccessPolicyRule{
								Application: "*",
								Namespace:   source.GetNamespace(),
								Cluster:     source.GetClusterName(),
							},
						},
					},
				},
			},
		},
	}
}
