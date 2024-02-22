package observability

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestOtelEndpointFromConfig(t *testing.T) {
	testCases := []struct {
		name             string
		collector        config.OtelCollector
		expectedEndpoint string
	}{
		{
			name: "TLS disabled",
			collector: config.OtelCollector{
				Tls:       false,
				Service:   "my-service",
				Namespace: "my-namespace",
				Port:      8080,
			},
			expectedEndpoint: "http://my-service.my-namespace:8080",
		},
		{
			name: "TLS enabled",
			collector: config.OtelCollector{
				Tls:       true,
				Service:   "my-service",
				Namespace: "my-namespace",
				Port:      8080,
			},
			expectedEndpoint: "https://my-service.my-namespace:8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualEndpoint := otelEndpointFromConfig(tc.collector)
			assert.Equal(t, tc.expectedEndpoint, actualEndpoint)
		})
	}
}

func TestOtelEnvVars(t *testing.T) {
	app := nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "my-namespace",
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Observability: &nais_io_v1.Observability{
				Tracing: &nais_io_v1.Tracing{
					Enabled: true,
				},
			},
		},
	}
	otel := config.Otel{
		Collector: config.OtelCollector{
			Tls:       false,
			Service:   "my-service",
			Namespace: "my-namespace",
			Port:      8080,
			Protocol:  "grcp",
		},
	}

	expectedEnvVars := []corev1.EnvVar{
		{
			Name:  "OTEL_SERVICE_NAME",
			Value: "my-app",
		},
		{
			Name:  "OTEL_RESOURCE_ATTRIBUTES",
			Value: "service.name=my-app,service.namespace=my-namespace",
		},
		{
			Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
			Value: "http://my-service.my-namespace:8080",
		},
		{
			Name:  "OTEL_EXPORTER_OTLP_PROTOCOL",
			Value: "grcp",
		},
		{
			Name:  "OTEL_EXPORTER_OTLP_INSECURE",
			Value: "true",
		},
	}

	actualEnvVars := otelEnvVars(&app, otel)

	assert.Equal(t, expectedEnvVars, actualEnvVars)
}
func TestLabelsFromCollectorConfig(t *testing.T) {
	testCases := []struct {
		name           string
		collector      config.OtelCollector
		expectedLabels map[string]string
	}{
		{
			name: "Valid labels",
			collector: config.OtelCollector{
				Labels: []string{"key1=value1", "key2=value2", "key3=value3"},
			},
			expectedLabels: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "Invalid labels",
			collector: config.OtelCollector{
				Labels: []string{"key1=value1", "key2=value2", "key3"},
			},
			expectedLabels: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "Empty labels",
			collector: config.OtelCollector{
				Labels: []string{},
			},
			expectedLabels: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualLabels := labelsFromCollectorConfig(tc.collector)
			assert.Equal(t, tc.expectedLabels, actualLabels)
		})
	}
}

func TestTracingNetpol(t *testing.T) {
	app := nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "my-namespace",
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Observability: &nais_io_v1.Observability{
				Tracing: &nais_io_v1.Tracing{
					Enabled: true,
				},
			},
		},
	}

	otel := config.Otel{
		Collector: config.OtelCollector{
			Labels:    []string{"key1=value1", "key2=value2", "key3=value3"},
			Namespace: "my-namespace",
			Port:      8080,
			Protocol:  "grcp",
			Service:   "my-service",
			Tls:       false,
		},
	}

	expectedName, err := namegen.ShortName(app.GetName()+"-"+"tracing", validation.DNS1035LabelMaxLength)
	assert.NoError(t, err)

	expectedObjectMeta := resource.CreateObjectMeta(&app)
	expectedObjectMeta.Name = expectedName

	expectedProtocolTCP := corev1.ProtocolTCP

	expectedNetworkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: expectedObjectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "my-app",
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &expectedProtocolTCP,
							Port:     &intstr.IntOrString{IntVal: 8080},
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"key1": "value1",
									"key2": "value2",
									"key3": "value3",
								},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "my-namespace",
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

	actualNetworkPolicy, err := tracingNetpol(&app, otel)
	assert.NoError(t, err)
	assert.Equal(t, expectedNetworkPolicy, actualNetworkPolicy)
}
