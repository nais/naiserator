package generator

import (
	"strings"
	"time"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/event"
	docker "github.com/novln/docker-parser"
)

func NewDeploymentEvent(app nais.Application) deployment.Event {
	image := ContainerImage(app.Spec.Image)

	return deployment.Event{
		Application:   app.Name,
		Cluster:       app.ClusterName,
		CorrelationID: app.Annotations[nais.CorrelationIDAnnotation],
		Environment:   environment(app),
		Image:         &image,
		Namespace:     app.Namespace,
		RolloutStatus: deployment.RolloutStatus_initialized,
		Source:        deployment.System_naiserator,
		Team:          app.Labels["team"],
		Timestamp:     time.Now().Unix(),
		Version:       image.Tag,
		Platform: &deployment.Platform{
			Type:    deployment.PlatformType_nais,
			Variant: "naiserator",
		},
	}
}

func environment(app nais.Application) deployment.Environment {
	if strings.HasPrefix(app.ClusterName, "prod-") {
		return deployment.Environment_production
	}
	return deployment.Environment_development
}

func hashtag(t string) (hash, tag string) {
	if strings.ContainsRune(t, ':') {
		return t, ""
	}
	return "", t
}

func ContainerImage(imageName string) deployment.ContainerImage {
	ref, err := docker.Parse(imageName)
	if err != nil {
		return deployment.ContainerImage{}
	}
	hash, tag := hashtag(ref.Tag())
	return deployment.ContainerImage{
		Name: ref.Repository(),
		Tag:  tag,
		Hash: hash,
	}
}
