package observability

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	resource.Source
	GetObservability() *nais_io_v1.Observability
}

// Standard environment variable name from https://opentelemetry.io/docs/specs/otel/protocol/exporter/
// Location is hard-coded because it is the same across installations, feel free to make it configurable.
const otelExporterEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
const collectorEndpoint = "http://tempo-distributor.nais-system:4317"

func envVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  otelExporterEndpoint,
			Value: collectorEndpoint,
		},
	}
}

func netpol(source Source) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": source.GetName(),
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/instance": "tempo",
								},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "nais-system",
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
		},
	}
}

func Create(source Source, ast *resource.Ast, _ any) error {
	obs := source.GetObservability()
	if obs == nil || obs.Tracing == nil || !obs.Tracing.Enabled {
		return nil
	}

	source.GetObjectMeta()
	ast.Env = append(ast.Env, envVars()...)
	ast.AppendOperation(resource.OperationCreateOrUpdate, netpol(source))

	return nil
}
