package observability

import (
	"fmt"
	"slices"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

type Source interface {
	resource.Source
	GetObservability() *nais_io_v1.Observability
}

type Config interface {
	GetObservability() config.Observability
}

// Standard environment variable names from https://opentelemetry.io/docs/specs/otel/protocol/exporter/
// These are hard-coded because they are the same across installations, feel free to make them configurable.
const otelServiceName = "OTEL_SERVICE_NAME"
const otelResourceAttributes = "OTEL_RESOURCE_ATTRIBUTES"
const otelExporterEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
const otelExporterProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
const otelExporterInsecure = "OTEL_EXPORTER_OTLP_INSECURE"

const logLabelDefault = "logs.nais.io/flow-default"
const logLabelPrefix = "logs.nais.io/flow-"

const autoInstrumentationInjectAnnotationPrefix = "instrumentation.opentelemetry.io/inject-"
const autoInstrumentationContainerNamesAnnotation = "instrumentation.opentelemetry.io/container-names"

func otelEndpointFromConfig(collector config.OtelCollector) string {
	schema := "http"
	if collector.Tls {
		schema = "https"
	}
	return fmt.Sprintf("%s://%s.%s:%d", schema, collector.Service, collector.Namespace, collector.Port)
}

func otelEnvVars(source Source, otel config.Otel) []corev1.EnvVar {
	collectorEndpoint := otelEndpointFromConfig(otel.Collector)
	collectorProtocol := otel.Collector.Protocol

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
		{
			Name:  otelExporterInsecure,
			Value: fmt.Sprintf("%t", !otel.Collector.Tls),
		},
	}
}

func otelAutoInstrumentAnnotations(source Source, otel config.Otel) map[string]string {
	runtime := source.GetObservability().AutoInstrumentation.Runtime
	autoInstrumentationInjectAnnotation := autoInstrumentationInjectAnnotationPrefix + runtime

	return map[string]string{
		autoInstrumentationInjectAnnotation:         otel.AutoInstrumentation.AppConfig,
		autoInstrumentationContainerNamesAnnotation: source.GetName(),
	}
}

func labelsFromCollectorConfig(collector config.OtelCollector) map[string]string {
	labels := collector.Labels
	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		kv := strings.Split(label, "=")
		if len(kv) != 2 {
			continue
		}
		labelMap[kv[0]] = kv[1]
	}
	return labelMap
}

func tracingNetpol(source Source, otel config.Otel) (*networkingv1.NetworkPolicy, error) {
	name, err := namegen.ShortName(source.GetName()+"-"+"tracing", validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	protocolTCP := corev1.ProtocolTCP
	collectorLabels := labelsFromCollectorConfig(otel.Collector)

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
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolTCP,
							Port:     &intstr.IntOrString{IntVal: int32(otel.Collector.Port)},
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: collectorLabels,
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": otel.Collector.Namespace,
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

func Create(source Source, ast *resource.Ast, config Config) error {
	obs := source.GetObservability()
	cfg := config.GetObservability()

	if obs == nil {
		return nil
	}

	if obs.AutoInstrumentation != nil && obs.AutoInstrumentation.Enabled && obs.Tracing != nil && obs.Tracing.Enabled {
		return fmt.Errorf("auto-instrumentation and tracing cannot be enabled at the same time")
	}

	if (obs.Tracing != nil && obs.Tracing.Enabled) || (obs.AutoInstrumentation != nil && obs.AutoInstrumentation.Enabled) {
		if !cfg.Otel.Enabled {
			return fmt.Errorf("tracing and auto-instrumentation are not supported in this cluster")
		}

		netpol, err := tracingNetpol(source, cfg.Otel)
		if err != nil {
			return err
		}

		if obs.AutoInstrumentation != nil && obs.AutoInstrumentation.Enabled {
			for k, v := range otelAutoInstrumentAnnotations(source, cfg.Otel) {
				ast.Annotations[k] = v
			}
		}

		ast.Env = append(ast.Env, otelEnvVars(source, cfg.Otel)...)
		ast.AppendOperation(resource.OperationCreateOrUpdate, netpol)
	}

	if obs.Logging != nil {
		if !obs.Logging.Enabled {
			ast.Labels[logLabelDefault] = "false"
			return nil
		}

		if len(obs.Logging.Destinations) > 0 {
			ast.Labels[logLabelDefault] = "false"

			for _, destination := range obs.Logging.Destinations {
				if !slices.Contains(cfg.Logging.Destinations, destination.ID) {
					return fmt.Errorf("logging destination %q does not exist in cluster", destination.ID)
				}

				ast.Labels[logLabelPrefix+destination.ID] = "true"
			}
		}
	}

	return nil
}
