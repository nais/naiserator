package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	types "github.com/nais/naiserator/pkg/event"
	log "github.com/sirupsen/logrus"
)

var (
	Deployment deployment
)

type deployment struct {
	EventChan chan *types.Event
	queueSize int
}

func init() {
	Deployment = deployment{
		EventChan: make(chan *types.Event),
		queueSize: 5,
	}
}

func setAppContext(e *types.Event, app *v1alpha1.Application) {
	e.RolloutStatus = types.RolloutStatus_initialized
	if e.Environment = types.Environment_development; strings.Contains(app.ClusterName, "prod") {
		e.Environment = types.Environment_production
	}

	e.Team = app.Labels["team"]
	e.Namespace = app.Namespace
	e.Cluster = app.ClusterName
	e.Application = app.Name

	parts := strings.SplitN(app.Spec.Image, ":", 2)
	image := types.ContainerImage{Name: parts[0], Tag: "latest"}
	if len(parts) != 1 {
		image.Tag = parts[1]
	}
	e.Image = &image

	e.Version = e.Image.GetTag() // Is this good enough?
}

func setIndividualContext(e *types.Event) {
	e.Timestamp = time.Now().Unix()
}

func DefaultEvent() *types.Event {
	id := "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

	UUID, err := uuid.NewRandom()
	if err != nil {
		log.Errorf("while creating a UUID string: %s", err)
	}
	id = UUID.String()
	platform := types.Platform{Type: types.PlatformType_nais, Variant: "naiserator"}

	return &types.Event{
		CorrelationID: id,
		Platform:      &platform,
		Source:        types.System_naiserator,
	}
}

func (deployment *deployment) InitializeAppRollout(ctx context.Context, app *v1alpha1.Application, logger *log.Entry) {
	e := DefaultEvent()
	setAppContext(e, app)
	setIndividualContext(e)

	e.RolloutStatus = types.RolloutStatus_initialized

	go func() {
		select {
		case deployment.EventChan <- e:
			logger.Info("successfully sent deployment Event(Initialized) to channel")
		case <-time.After(1 * time.Minute):
			logger.Errorf("while waiting to send deployment Event to channel: %q", e)
			// Reconnect logic?
		case <-ctx.Done():
			logger.Infof("Cancelled before getting to send deployment Event(Initialized) to channel: %s", ctx.Err())
		}
	}()
}

func (deployment *deployment) WaitForApplicationRollout(ctx context.Context, app *v1alpha1.Application, logger *log.Entry, ready <-chan bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	e := DefaultEvent()
	setAppContext(e, app)
	setIndividualContext(e)
	for {
		select {
		case <-ready:
			e.RolloutStatus = types.RolloutStatus_complete
			select {
			case deployment.EventChan <- e:
				logger.Info("successfully sent deployment Event(Successful) to channel")
			case <-time.After(1 * time.Minute):
				logger.Errorf("while waiting to send deployment Event to channel: %q", e)
				// Reconnect logic?
			}
			return
		case <-ctx.Done():
			logger.Infof("Cancelled before getting to send deployment event(Successful) to channel: %s", ctx.Err())
			//send RolouStatus_unknown here?
			return
		}
	}
}

func (client *Client) ProducerLoop() {
	for {
		select {
		case event := <-Deployment.EventChan:
			if err := client.sendDeploymentEvent(event); err != nil {
				log.Errorf("while sending deployment event to kafka: %s", err)
			}
		}
	}
}

func (client *Client) sendDeploymentEvent(event *types.Event) error {
	payload, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("while encoding Protobuf: %s", err)
	}
	reply := &sarama.ProducerMessage{
		Topic:     client.ProducerTopic,
		Timestamp: time.Now(),
		Value:     sarama.StringEncoder(payload),
	}

	_, offset, err := client.Producer.SendMessage(reply)
	if err != nil {
		return fmt.Errorf("while sending reply over Kafka: %s", err)
	}

	log.WithFields(log.Fields{
		"kafka_offset":    offset,
		"kafka_timestamp": reply.Timestamp,
		"kafka_topic":     reply.Topic,
	}).Infof("Deployment event sent successfully")

	return nil
}
