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

func otelAttributes(name, namesspace string, env []corev1.EnvVar, destinations []string) string {
	reservedAttributes := map[string]bool{
		"service.name":      true,
		"service.namespace": true,
		"nais.backend":      true,
	}

	attributes := fmt.Sprintf("service.name=%s,service.namespace=%s", name, namesspace)

	if len(destinations) > 0 {
		slices.Sort(destinations)
		attributes = fmt.Sprintf("%s,nais.backend=%s", attributes, strings.Join(destinations, ";"))
	}

	// Set user-defined attributes without overwriting default attributes
	for _, e := range env {
		if e.Name == otelResourceAttributes {
			for _, kv := range strings.Split(e.Value, ",") {
				parts := strings.Split(kv, "=")
				if len(parts) == 2 && !reservedAttributes[parts[0]] {
					attributes = fmt.Sprintf("%s,%s", attributes, kv)
				}
			}
		}
	}

	return attributes
}

func otelFilterEnvVars(env []corev1.EnvVar) []corev1.EnvVar {
	filtered := []corev1.EnvVar{}
	for _, e := range env {
		if e.Name != otelServiceName &&
			e.Name != otelResourceAttributes &&
			e.Name != otelExporterEndpoint &&
			e.Name != otelExporterProtocol &&
			e.Name != otelExporterInsecure {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func OtelEnvVars(name, namespace string, env []corev1.EnvVar, destinations []string, otel config.Otel) []corev1.EnvVar {
	return append(otelFilterEnvVars(env), []corev1.EnvVar{
		{
			Name:  otelServiceName,
			Value: name,
		},
		{
			Name:  otelResourceAttributes,
			Value: otelAttributes(name, namespace, env, destinations),
		},
		{
			Name:  otelExporterEndpoint,
			Value: otelEndpointFromConfig(otel.Collector),
		},
		{
			Name:  otelExporterProtocol,
			Value: otel.Collector.Protocol,
		},
		{
			Name:  otelExporterInsecure,
			Value: fmt.Sprintf("%t", !otel.Collector.Tls),
		},
	}...)
}

func otelAutoInstrumentationDestinations(source Source, otel config.Otel) ([]string, error) {
	destinations := source.GetObservability().AutoInstrumentation.Destinations
	destinationIDs := make([]string, len(destinations))

	for i, destination := range destinations {
		destinationIDs[i] = destination.ID
		if !slices.Contains(otel.Destinations, destination.ID) {
			return nil, fmt.Errorf("auto-instrumentation destination %q does not exist in cluster", destination.ID)
		}
	}

	return destinationIDs, nil
}

func otelAutoInstrumentAnnotations(source Source, otel config.Otel) map[string]string {
	runtime := source.GetObservability().AutoInstrumentation.Runtime
	autoInstrumentationInjectAnnotation := autoInstrumentationInjectAnnotationPrefix + runtime

	return map[string]string{
		autoInstrumentationInjectAnnotation:         otel.AutoInstrumentation.AppConfig,
		autoInstrumentationContainerNamesAnnotation: source.GetName(),
	}
}

func netpolLabelsFromCollectorConfig(collector config.OtelCollector) map[string]string {
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

func logLabels(obs *nais_io_v1.Observability, cfg config.Logging) (map[string]string, error) {
	if !obs.Logging.Enabled {
		return map[string]string{logLabelDefault: "false"}, nil
	}

	labels := map[string]string{}

	if len(obs.Logging.Destinations) > 0 {
		labels[logLabelDefault] = "false"

		for _, destination := range obs.Logging.Destinations {
			if !slices.Contains(cfg.Destinations, destination.ID) {
				return nil, fmt.Errorf("logging destination %q does not exist in cluster", destination.ID)
			}

			labels[logLabelPrefix+destination.ID] = "true"
		}
	}

	return labels, nil
}

func otelNetpol(source Source, otel config.Otel) (*networkingv1.NetworkPolicy, error) {
	name, err := namegen.ShortName(source.GetName()+"-"+"tracing", validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	protocolTCP := corev1.ProtocolTCP
	collectorLabels := netpolLabelsFromCollectorConfig(otel.Collector)

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
			return fmt.Errorf("opentelemetry is not supported for this cluster")
		}

		if !cfg.Otel.AutoInstrumentation.Enabled && obs.AutoInstrumentation != nil && obs.AutoInstrumentation.Enabled {
			return fmt.Errorf("auto-instrumentation is not supported for this cluster")
		}

		netpol, err := otelNetpol(source, cfg.Otel)
		if err != nil {
			return err
		}

		destinations := []string{}
		if obs.AutoInstrumentation != nil && obs.AutoInstrumentation.Enabled {
			destinations, err = otelAutoInstrumentationDestinations(source, cfg.Otel)
			if err != nil {
				return err
			}

			for k, v := range otelAutoInstrumentAnnotations(source, cfg.Otel) {
				ast.Annotations[k] = v
			}
		}

		ast.Env = OtelEnvVars(source.GetName(), source.GetNamespace(), ast.Env, destinations, cfg.Otel)
		ast.AppendOperation(resource.OperationCreateOrUpdate, netpol)
	}

	if obs.Logging != nil {
		logLabels, err := logLabels(obs, cfg.Logging)
		if err != nil {
			return err
		}

		for k, v := range logLabels {
			ast.Labels[k] = v
		}
	}

	return nil
}
