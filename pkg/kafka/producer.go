package kafka

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/protobuf/proto"
	"github.com/nais/naiserator/pkg/event"
	log "github.com/sirupsen/logrus"
)

var (
	// Send deployment events here to dispatch them to Kafka.
	Events = make(chan deployment.Event, 4096)
)

func (client *Client) ProducerLoop() {
	for event := range Events {
		if err := client.send(event); err != nil {
			log.Errorf("while sending deployment event to kafka: %s", err)
		}
	}
}

func (client *Client) send(event deployment.Event) error {
	payload, err := proto.Marshal(&event)
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
