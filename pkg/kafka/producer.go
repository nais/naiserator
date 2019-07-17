package kafka

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/protobuf/proto"
	"github.com/nais/naiserator/pkg/event"
	log "github.com/sirupsen/logrus"
)

type Message struct {
	Event  deployment.Event
	Logger log.Entry
}

var (
	// Send deployment events here to dispatch them to Kafka.
	Events = make(chan Message, 4096)
)

func (client *Client) ProducerLoop() {
	for message := range Events {
		if err := client.send(message); err != nil {
			log.Errorf("while sending deployment event to kafka: %s", err)
		}
	}
}

func (client *Client) send(message Message) error {
	payload, err := proto.Marshal(&message.Event)
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

	message.Logger.WithFields(log.Fields{
		"kafka_offset":    offset,
		"kafka_timestamp": reply.Timestamp,
		"kafka_topic":     reply.Topic,
	}).Infof("Deployment event sent successfully")

	return nil
}
