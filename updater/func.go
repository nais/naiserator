package updater

import (
	"context"

	nais_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOrUpdate(ctx context.Context, client client.Client, resource runtime.Object) func() error {
	return func() error {
		log.Infof("creating new %s", resource)
		err := client.Create(ctx, resource)
		if errors.IsAlreadyExists(err) {
			err = client.Update(ctx, resource)
		}
		return err
	}
}

func CreateOrRecreate(ctx context.Context, client client.Client, resource runtime.Object) func() error {
	return func() error {
		log.Infof("pre-deleting %s", resource)
		err := client.Delete(ctx, resource)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		log.Infof("creating new %s", resource)
		return client.Create(ctx, resource)
	}
}

func CreateIfNotExists(ctx context.Context, client client.Client, resource runtime.Object) func() error {
	return func() error {
		log.Infof("creating new %s", resource)
		err := client.Create(ctx, resource)
		if err != nil && errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
}

func DeleteIfExists(ctx context.Context, client client.Client, resource runtime.Object) func() error {
	return func() error {
		log.Infof("creating new %s", resource)
		err := client.Delete(ctx, resource)
		if err != nil && errors.IsNotFound(err) {
			return nil
		}
		return err
	}
}

func FindAll(ctx context.Context, client client.Client, app *nais_v1alpha1.Application) ([]runtime.Object, error) {
	panic("not implemented")

	/*
		{
			c := clientSet.CoreV1().Services(app.Namespace)
			existing, err := c.List(metav1.ListOptions{LabelSelector: "app=" + app.Name})
			if err != nil && !errors.IsNotFound(err) {
				return nil, fmt.Errorf("discover %s: %s", "*corev1.Service", err)
			} else if existing != nil {
				items, err := meta.ExtractList(existing)
				if err != nil {
					return nil, fmt.Errorf("extract list of %s: %s", "*corev1.Service", err)
				}
				resources = append(resources, items...)
			}
		}
				return withOwnerReference(app, resources), nil
	*/

}

func withOwnerReference(app *nais_v1alpha1.Application, resources []runtime.Object) []runtime.Object {
	owned := make([]runtime.Object, 0, len(resources))

	hasOwnerReference := func(r runtime.Object) (bool, error) {
		m, err := meta.Accessor(r)
		if err != nil {
			return false, err
		}
		for _, ref := range m.GetOwnerReferences() {
			if ref.UID == app.UID {
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
