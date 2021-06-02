package resource

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	metav1.Object
	metav1.ObjectMetaAccessor
	CreateObjectMeta() metav1.ObjectMeta
	CreateObjectMetaWithName(string) metav1.ObjectMeta
	CreateAppNamespaceHash() string
	CorrelationID() string
	SkipDeploymentMessage() bool
	CreateEvent(string, string, string) *corev1.Event
}
