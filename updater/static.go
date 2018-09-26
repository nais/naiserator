package updater

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CopyMeta copies resource metadata from one resource to another.
// used when updating existing resources in the cluster.
func CopyMeta(src, dst metav1.ObjectMetaAccessor) {
	dst.GetObjectMeta().SetResourceVersion(src.GetObjectMeta().GetResourceVersion())
}

// ClusterIP must be retained as the field is immutable.
func CopyService(src, dst *corev1.Service) {
	dst.Spec.ClusterIP = src.Spec.ClusterIP
}
