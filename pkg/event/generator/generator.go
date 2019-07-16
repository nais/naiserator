package generator

import (
	"strings"
	"time"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/event"
)

func NewDeploymentEvent(app nais.Application) deployment.Event {
	image := containerImage(app.Spec.Image)

	return deployment.Event{
		Application:   app.Name,
		Cluster:       app.ClusterName,
		CorrelationID: app.Annotations["nais.io/deployment-correlation-id"],
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

func containerImage(imageName string) deployment.ContainerImage {
	parts := strings.SplitN(imageName, ":", 2)
	tag := "latest"
	if len(parts) != 1 {
		tag = parts[1]
	}
	return deployment.ContainerImage{
		Name: parts[0],
		Tag:  tag,
	}
}
