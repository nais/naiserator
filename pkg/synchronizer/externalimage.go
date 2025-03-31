package synchronizer

import (
	"context"
	"errors"
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ImageNotFound = errors.New("external image resource not found")

type ImageSource interface {
	resource.Source
	GetImage() string
	GetEffectiveImage() string
}

func getExternalImage(ctx context.Context, source ImageSource, kube client.Client) (string, error) {
	key := client.ObjectKey{
		Name:      source.GetName(),
		Namespace: source.GetNamespace(),
	}
	image := &nais_io_v1.Image{}
	err := kube.Get(ctx, key, image)
	if err != nil {
		if !k8s_errors.IsNotFound(err) {
			return "", fmt.Errorf("query existing image: %s", err)
		}
		return "", fmt.Errorf("%s: %w", key, ImageNotFound)
	}
	return image.Spec.Image, nil
}

func imageHasChanged(wantedImage string, source ImageSource) bool {
	// Avoid resync if effective image is not set, and image is set on spec, as that will be the case for every app on first sync of this feature
	// Remove once a majority of apps have effective image set
	if len(source.GetEffectiveImage()) == 0 && len(source.GetImage()) > 0 {
		return false
	}
	return wantedImage != source.GetEffectiveImage()
}

func getWantedImage(ctx context.Context, source ImageSource, kube client.Client) (string, error) {
	wantedImage := source.GetImage()
	if len(wantedImage) == 0 {
		externalImage, err := getExternalImage(ctx, source, kube)
		if err != nil {
			return "", err
		}
		wantedImage = externalImage
	}
	return wantedImage, nil
}

func updateEffectiveImage(source resource.Source, wantedImage string) {
	status := source.GetStatus()
	status.EffectiveImage = wantedImage
}
