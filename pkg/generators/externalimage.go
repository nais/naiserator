package generators

import (
	"context"
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SynchronizedImage interface {
	resource.Source
	GetImage() string
}

var _ SynchronizedImage = &nais_io_v1.Naisjob{}
var _ SynchronizedImage = &nais_io_v1alpha1.Application{}

func SetSynchronizedImage(ctx context.Context, source SynchronizedImage, key client.ObjectKey, kube client.Client) error {
	imageTag := source.GetImage()
	status := source.GetStatus()
	if len(imageTag) > 0 {
		status.SynchronizedImage = imageTag
		return nil
	}

	image := &nais_io_v1.Image{}
	err := kube.Get(ctx, key, image)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("query existing image: %s", err)
		}
		return nil
	}
	status.SynchronizedImage = image.Spec.Image
	return nil
}
