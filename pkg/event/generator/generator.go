package generator

import (
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/naiserator/config"
	docker "github.com/novln/docker-parser"
	"github.com/spf13/viper"
)

func NewDeploymentEvent(app nais.Application) deployment.Event {
	image := ContainerImage(app.Spec.Image)
	ts := convertTimestamp(time.Now())

	return deployment.Event{
		CorrelationID: app.Status.CorrelationID,
		Platform: &deployment.Platform{
			Type: deployment.PlatformType_nais,
		},
		Source:          deployment.System_naiserator,
		Deployer:        nil,
		Team:            app.Labels["team"],
		RolloutStatus:   deployment.RolloutStatus_initialized,
		Environment:     environment(),
		SkyaEnvironment: "",
		Namespace:       app.Namespace,
		Cluster:         viper.GetString(config.ClusterName),
		Application:     app.Name,
		Version:         image.GetTag(),
		Image:           &image,
		Timestamp:       &ts,
	}
}

func convertTimestamp(t time.Time) timestamp.Timestamp {
	return timestamp.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.UnixNano()),
	}
}

func environment() deployment.Environment {
	if strings.HasPrefix(viper.GetString(config.ClusterName), "prod-") {
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
