package kafka

import (
	"fmt"
	"math/rand"
	"os"

	flag "github.com/spf13/pflag"
)

type SASL struct {
	Enabled   bool   `json:"enabled"`
	Handshake bool   `json:"handshake"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type TLS struct {
	Enabled  bool `json:"enabled"`
	Insecure bool `json:"insecure"`
}

type Config struct {
	Enabled      bool   `json:"enabled"`
	Brokers      string `json:"brokers"`
	Topic        string `json:"topic"`
	ClientID     string `json:"client-id"`
	GroupID      string `json:"group-id"`
	LogVerbosity string `json:"log-verbosity"`
	TLS          TLS    `json:"tls"`
	SASL         SASL   `json:"sasl"`
}

func DefaultGroupName() string {
	if hostname, err := os.Hostname(); err == nil {
		return fmt.Sprintf("naiserator-%s", hostname)
	}
	return fmt.Sprintf("naiserator-%d", rand.Int())
}

func SetupFlags() {
	defaultGroup := DefaultGroupName()
	flag.StringSlice("kafka.brokers", []string{"localhost:9092"}, "Comma-separated list of Kafka brokers, HOST:PORT.")
	flag.String("kafka.topic", "deploymentEvents", "Kafka topic for deployment status.")
	flag.String("kafka.client-id", defaultGroup, "Kafka client ID.")
	flag.String("kafka.group-id", defaultGroup, "Kafka consumer group ID.")
	flag.String("kafka.log-verbosity", "trace", "Log verbosity for Kafka client.")
	flag.Bool("kafka.sasl.enabled", false, "Enable SASL authentication.")
	flag.Bool("kafka.sasl.handshake", true, "Use handshake for SASL authentication.")
	flag.String("kafka.sasl.username", "", "Username for Kafka authentication.")
	flag.String("kafka.sasl.password", "", "Password for Kafka authentication.")
	flag.Bool("kafka.tls.enabled", false, "Use TLS for connecting to Kafka.")
	flag.Bool("kafka.tls.insecure", false, "Allow insecure Kafka TLS connections.")
	flag.Bool("kafka.enabled", false, "Enable connection to kafka")
}
