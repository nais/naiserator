package resource

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	metav1.Object
	metav1.ObjectMetaAccessor
	metav1.Common
	CreateAppNamespaceHash() string
	GetOwnerReference() metav1.OwnerReference
	CorrelationID() string
	SkipDeploymentMessage() bool
	CreateEvent(string, string, string) *corev1.Event
	LogFields() log.Fields
}

func CreateObjectMeta(source Source) metav1.ObjectMeta {
	labels := map[string]string{}

	for k, v := range source.GetLabels() {
		labels[k] = v
	}

	labels["app"] = source.GetName()

	return metav1.ObjectMeta{
		Name:      source.GetName(),
		Namespace: source.GetNamespace(),
		Labels:    labels,
		Annotations: map[string]string{
			nais_io_v1.DeploymentCorrelationIDAnnotation: source.CorrelationID(),
		},
		OwnerReferences: []metav1.OwnerReference{source.GetOwnerReference()},
	}
}
