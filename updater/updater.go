package updater

import (
	"context"
	"fmt"
	"time"

	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AnnotateIfExists copies annotations of the given resource into the existing resource.
// No other parts of the existing resource is touched.
func AnnotateIfExists(ctx context.Context, cli client.Client, scheme *runtime.Scheme, annotationSource client.Object) func() error {
	return func() error {
		log.Infof("AnnotateIfExists %s", liberator_scheme.TypeName(annotationSource))
		existing, err := scheme.New(annotationSource.GetObjectKind().GroupVersionKind())
		if err != nil {
			return fmt.Errorf("internal error: %w", err)
		}
		objectKey := client.ObjectKeyFromObject(annotationSource)

		err = cli.Get(ctx, objectKey, existing.(client.Object))
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			} else {
				return fmt.Errorf("get for AnnotateIfExists: %w", err)
			}
		}

		original := existing.DeepCopyObject().(client.Object)
		patchSource := client.MergeFrom(original)

		modified := existing.(client.Object)
		CopyAnnotations(modified, annotationSource)

		err = cli.Patch(ctx, modified, patchSource)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("patch for AnnotateIfExists: %w", err)
		}
		return nil
	}
}

func CreateOrUpdate(ctx context.Context, cli client.Client, scheme *runtime.Scheme, resource client.Object) func() error {
	return func() error {
		log.Infof("CreateOrUpdate %s", liberator_scheme.TypeName(resource))
		existing, err := scheme.New(resource.GetObjectKind().GroupVersionKind())
		if err != nil {
			return fmt.Errorf("internal error: %w", err)
		}
		objectKey := client.ObjectKeyFromObject(resource)

		namespace := corev1.Namespace{}
		err = cli.Get(ctx, client.ObjectKey{Name: resource.GetNamespace()}, &namespace)
		if err != nil {
			return fmt.Errorf("get namespace %s: %w", resource.GetNamespace(), err)
		}

		isPgNamespace := namespace.Labels["nais.io/type"] == "postgres"

		err = cli.Get(ctx, objectKey, existing.(client.Object))

		if errors.IsNotFound(err) {
			err = cli.Create(ctx, resource)
		} else if err == nil {
			err = CopyMeta(resource, existing)
			if err != nil {
				return err
			}
			err = CopyImmutable(resource, existing)
			if err != nil {
				return err
			}
			err = AssertValidOwnerReference(resource, existing, isPgNamespace)
			if err != nil {
				return err
			}
			err = KeepOwnerReference(resource, existing)
			if err != nil {
				return err
			}
			err = cli.Update(ctx, resource)
		}

		if err != nil {
			return err
		}
		return err
	}
}

func CreateOrRecreate(ctx context.Context, cli client.Client, scheme *runtime.Scheme, resource client.Object) func() error {
	return func() error {
		log.Infof("CreateOrRecreate %s", liberator_scheme.TypeName(resource))
		deleteOptions := &client.DeleteOptions{}
		client.PropagationPolicy(metav1.DeletePropagationBackground).ApplyToDelete(deleteOptions)
		err := cli.Delete(ctx, resource, deleteOptions)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		existing, err := scheme.New(resource.GetObjectKind().GroupVersionKind())
		if err != nil {
			return fmt.Errorf("internal error: %w", err)
		}
		objectKey := client.ObjectKeyFromObject(resource)
		timedOut := time.Now().Add(time.Minute + time.Duration(5))

		for {
			err = cli.Get(ctx, objectKey, existing.(client.Object))
			if errors.IsNotFound(err) {
				break
			}
			if err != nil {
				return fmt.Errorf("internal error: %v", err)
			}
			if time.Now().After(timedOut) {
				return fmt.Errorf("timed out waiting for deletion of %v/%v", resource.GetObjectKind(), resource.GetName())
			}
			time.Sleep(1 * time.Second)
		}

		return cli.Create(ctx, resource)
	}
}

func CreateIfNotExists(ctx context.Context, cli client.Client, resource client.Object) func() error {
	return func() error {
		log.Infof("CreateIfNotExists %s", liberator_scheme.TypeName(resource))
		err := cli.Create(ctx, resource)
		if err != nil && errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
}

func DeleteIfExists(ctx context.Context, cli client.Client, resource client.Object) func() error {
	return func() error {
		log.Infof("DeleteIfExists %s", liberator_scheme.TypeName(resource))
		err := cli.Delete(ctx, resource)
		if err != nil && errors.IsNotFound(err) {
			return nil
		}
		return err
	}
}

// FindAll finds all Kubernetes resource matching label selector 'app=NAME' for all specified types
func FindAll(ctx context.Context, cli client.Client, types []client.ObjectList, source resource.Source) ([]runtime.Object, error) {
	// Set up label selector 'app=NAME'
	labelSelector := labels.NewSelector()
	labelreq, err := labels.NewRequirement("app", selection.Equals, []string{source.GetName()})
	if err != nil {
		return nil, err
	}
	labelSelector = labelSelector.Add(*labelreq)
	listopt := &client.ListOptions{
		LabelSelector: labelSelector,
	}

	resources := make([]runtime.Object, 0)

	for _, obj := range types {
		err = cli.List(ctx, obj, listopt)
		if err != nil {
			return nil, fmt.Errorf("list %T: %w", obj, err)
		}

		_ = meta.EachListItem(obj, func(item runtime.Object) error {
			resources = append(resources, item)
			return nil
		})

		if panicErr := recover(); panicErr != nil {
			return nil, fmt.Errorf("recovered PANIC when listing %T: %s", obj, panicErr)
		}
	}

	return withOwnerReference(source, resources), nil
}

// KeepOwnerReference ensures that if ownerReference is set on the source object,
// it is copied to the destination object.
//
// Otherwise, it doesn't touch the destination object.
func KeepOwnerReference(dst, src runtime.Object) error {
	srcacc, err := meta.Accessor(src)
	if err != nil {
		return err
	}

	dstacc, err := meta.Accessor(dst)
	if err != nil {
		return err
	}

	existingReferences := srcacc.GetOwnerReferences()

	if len(existingReferences) > 0 {
		dstacc.SetOwnerReferences(existingReferences)
	}

	return nil
}

func ownerReferenceSimilar(a, b metav1.OwnerReference) bool {
	return a.Name == b.Name && a.Kind == b.Kind
}

func AssertValidOwnerReference(dst, src runtime.Object, isPgNamespace bool) error {
	srcacc, err := meta.Accessor(src)
	if err != nil {
		return err
	}

	dstacc, err := meta.Accessor(dst)
	if err != nil {
		return err
	}

	existingReferences := srcacc.GetOwnerReferences()
	newReferences := dstacc.GetOwnerReferences()

	// If the destination resource is in a Postgres namespace, we do not want ownerReferences.
	// Users are not able to create resources in the Postgres namespace, so we can assume everything
	// in that namespace belongs to the platform.
	if newReferences == nil && isPgNamespace {
		return nil
	}

	// Resources with no ownerReference will not be touched, in case it was created manually.
	//
	// Iterate through all combinations, and if resource is owned by the same Application/Naisjob that triggered
	// the creation, it should be allowed. Otherwise, reject it.
	for _, dstRef := range newReferences {
		for _, srcRef := range existingReferences {
			if ownerReferenceSimilar(srcRef, dstRef) {
				return nil
			}
		}
	}

	// TODO: remove this; this is a temporary fix for adoption of restored resources from velero.
	if len(existingReferences) == 0 {
		log.Warnf("Existing resource %s has no ownerReferences, but new resource has ownerReferences set. Adopting resource...", liberator_scheme.TypeName(dst))
		return nil
	}

	return fmt.Errorf("refusing to overwrite manually edited resource; please add the correct ownerReference in order to continue")
}

// CopyMeta copies resource metadata from one resource to another.
// used when updating existing resources in the cluster.
func CopyMeta(dst, src runtime.Object) error {
	srcacc, err := meta.Accessor(src)
	if err != nil {
		return err
	}

	dstacc, err := meta.Accessor(dst)
	if err != nil {
		return err
	}

	// Must always be present when updating a resource
	dstacc.SetResourceVersion(srcacc.GetResourceVersion())
	dstacc.SetUID(srcacc.GetUID())
	dstacc.SetSelfLink(srcacc.GetSelfLink())

	return err
}

func CopyCNRM(dst, src metav1.Object) {
	CopyAnnotation(dst, src, "cnrm.cloud.google.com/state-into-spec")
}

func CopyAnnotation(dst, src metav1.Object, key string) {
	anno := dst.GetAnnotations()
	if anno == nil {
		anno = make(map[string]string)
	}
	v := src.GetAnnotations()[key]
	if len(v) > 0 {
		anno[key] = v
	}
	dst.SetAnnotations(anno)
}

func CopyAnnotations(dst, src metav1.Object) {
	anno := dst.GetAnnotations()
	if anno == nil {
		anno = make(map[string]string)
	}
	for key, value := range src.GetAnnotations() {
		anno[key] = value
	}
	dst.SetAnnotations(anno)
}

func CopyImmutable(dst, src runtime.Object) error {
	switch srcTyped := src.(type) {
	case *corev1.Service:
		// ClusterIP must be retained as the field is immutable.
		dstTyped, ok := dst.(*corev1.Service)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		dstTyped.Spec.ClusterIP = srcTyped.Spec.ClusterIP

	case *sql_cnrm_cloud_google_com_v1beta1.SQLInstance:
		dstTyped, ok := dst.(*sql_cnrm_cloud_google_com_v1beta1.SQLInstance)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		CopyCNRM(dstTyped, srcTyped)
		dstTyped.Spec.ResourceID = srcTyped.Spec.ResourceID

	case *sql_cnrm_cloud_google_com_v1beta1.SQLDatabase:
		dstTyped, ok := dst.(*sql_cnrm_cloud_google_com_v1beta1.SQLDatabase)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		CopyCNRM(dstTyped, srcTyped)
		dstTyped.Spec.ResourceID = srcTyped.Spec.ResourceID

	case *sql_cnrm_cloud_google_com_v1beta1.SQLUser:
		dstTyped, ok := dst.(*sql_cnrm_cloud_google_com_v1beta1.SQLUser)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		CopyCNRM(dstTyped, srcTyped)
		dstTyped.Spec.ResourceID = srcTyped.Spec.ResourceID

	case *sql_cnrm_cloud_google_com_v1beta1.SQLSSLCert:
		dstTyped, ok := dst.(*sql_cnrm_cloud_google_com_v1beta1.SQLSSLCert)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		CopyCNRM(dstTyped, srcTyped)
		dstTyped.Spec.ResourceID = srcTyped.Spec.ResourceID

	case *storage_cnrm_cloud_google_com_v1beta1.StorageBucket:
		dstTyped, ok := dst.(*storage_cnrm_cloud_google_com_v1beta1.StorageBucket)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		CopyCNRM(dstTyped, srcTyped)
		dstTyped.Spec.ResourceID = srcTyped.Spec.ResourceID

	case *storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControl:
		dstTyped, ok := dst.(*storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControl)
		if !ok {
			return fmt.Errorf("source and destination types differ (%T != %T)", src, dst)
		}
		CopyCNRM(dstTyped, srcTyped)
	}

	return nil
}

func withOwnerReference(source resource.Source, resources []runtime.Object) []runtime.Object {
	owned := make([]runtime.Object, 0, len(resources))

	hasOwnerReference := func(r runtime.Object) (bool, error) {
		m, err := meta.Accessor(r)
		if err != nil {
			return false, err
		}
		for _, ref := range m.GetOwnerReferences() {
			if ref.UID == source.GetUID() {
				return true, nil
			}
		}
		return false, nil
	}

	for _, resource := range resources {
		ok, err := hasOwnerReference(resource)
		if err == nil && ok {
			owned = append(owned, resource)
		}
	}

	return owned
}
