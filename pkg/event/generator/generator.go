package generator

import (
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	docker "github.com/novln/docker-parser"
	"github.com/spf13/viper"
)

type ImageSource interface {
	resource.Source
	GetImage() string
}

func NewDeploymentEvent(source ImageSource) *deployment.Event {
	image := ContainerImage(source.GetImage())
	ts := convertTimestamp(time.Now())

	return &deployment.Event{
		CorrelationID: source.CorrelationID(),
		Platform: &deployment.Platform{
			Type: deployment.PlatformType_nais,
		},
		Source:          deployment.System_naiserator,
		Deployer:        nil,
		Team:            source.GetLabels()["team"],
		RolloutStatus:   deployment.RolloutStatus_initialized,
		Environment:     environment(),
		SkyaEnvironment: "",
		Namespace:       source.GetNamespace(),
		Cluster:         viper.GetString(config.ClusterName),
		Application:     source.GetName(),
		Version:         image.GetTag(),
		Image:           &image,
		Timestamp:       &ts,
		GitCommitSha:    source.GetAnnotations()["deploy.nais.io/github-sha"],
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
