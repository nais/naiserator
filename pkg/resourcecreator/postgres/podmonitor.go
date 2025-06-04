package postgres

import (
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	pov1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPodMonitor(source Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.OwnerReferences = nil
	objectMeta.Name = pgClusterName
	objectMeta.Namespace = pgNamespace

	podMonitor := &pov1.PodMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodMonitor",
			APIVersion: pov1.SchemeGroupVersion.String(),
		},
		ObjectMeta: objectMeta,
		Spec: pov1.PodMonitorSpec{
			JobLabel:        "app.kubernetes.io/name",
			PodTargetLabels: []string{"app", "cluster-name", "team"},
			PodMetricsEndpoints: []pov1.PodMetricsEndpoint{
				{
					Port:        "exporter",
					Path:        "/metrics",
					HonorLabels: false,
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"application": "spilo", "cluster-name": pgClusterName},
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, podMonitor)
}
