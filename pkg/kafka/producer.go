package kafka

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

type Message []byte

type Interface interface {
	Produce(msg Message) (int64, error)
}

func New(brokers []string, topic string, tlsConfig *tls.Config, logger *log.Logger) (*Producer, error) {
	config := sarama.NewConfig()
	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true
	config.ClientID, _ = os.Hostname()
	sarama.Logger = logger

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

func (p *Producer) Produce(msg Message) (offset int64, err error) {
	producerMessage := &sarama.ProducerMessage{
		Topic:     p.topic,
		Value:     sarama.ByteEncoder(msg),
		Timestamp: time.Now(),
	}

	_, offset, err = p.producer.SendMessage(producerMessage)

	return
}
