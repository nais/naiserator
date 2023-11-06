package observability

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Source interface {
	resource.Source
	GetObservability() *nais_io_v1.Observability
}

// Standard environment variable names from https://opentelemetry.io/docs/specs/otel/protocol/exporter/
// These are hard-coded because they are the same across installations, feel free to make them configurable.
const otelServiceName = "OTEL_SERVICE_NAME"
const otelResourceAttributes = "OTEL_RESOURCE_ATTRIBUTES"
const otelExporterEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
const collectorEndpoint = "http://tempo-distributor.nais-system:4317"
const otelExporterProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
const collectorProtocol = "grpc"

func envVars(source Source) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  otelServiceName,
			Value: source.GetName(),
		},
		{
			Name:  otelResourceAttributes,
			Value: fmt.Sprintf("service.name=%s,service.namespace=%s", source.GetName(), source.GetNamespace()),
		},
		{
			Name:  otelExporterEndpoint,
			Value: collectorEndpoint,
		},
		{
			Name:  otelExporterProtocol,
			Value: collectorProtocol,
		},
	}
}

func netpol(source Source) (*networkingv1.NetworkPolicy, error) {
	name, err := namegen.ShortName(source.GetName()+"-"+"tracing", validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	return &networkingv1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
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
									"kubernetes.io/metadata.name": "nais-system",
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
	}, nil
}

func Create(source Source, ast *resource.Ast, _ any) error {
	obs := source.GetObservability()
	if obs == nil || obs.Tracing == nil || !obs.Tracing.Enabled {
		return nil
	}

	np, err := netpol(source)
	if err != nil {
		return err
	}

	ast.Env = append(ast.Env, envVars(source)...)
	ast.AppendOperation(resource.OperationCreateOrUpdate, np)

	return nil
}
