package updater

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// CopyMeta copies resource metadata from one resource to another.
// used when updating existing resources in the cluster.
func CopyMeta(src, dst runtime.Object) error {
	srcacc, err := meta.Accessor(src)
	if err != nil {
		return err
	}

	dstacc, err := meta.Accessor(dst)
	if err != nil {
		return err
	}

	dstacc.SetResourceVersion(srcacc.GetResourceVersion())
	dstacc.SetUID(srcacc.GetUID())
	dstacc.SetSelfLink(srcacc.GetSelfLink())

	return err
}

func CopyImmutable(src, dst runtime.Object) error {
	switch srcTyped := src.(type) {
	case *corev1.Service:
		// ClusterIP must be retained as the field is immutable.
		dstTyped, ok := dst.(*corev1.Service)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		dstTyped.Spec.ClusterIP = srcTyped.Spec.ClusterIP
	}
	return nil
}
