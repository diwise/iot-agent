package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

type Client interface {
	Start() error
	Stop()
}

type Config struct {
	enabled   bool
	host      string
	keepAlive int64
	user      string
	password  string
	topics    []string
	clientId  string
}

func NewClient(ctx context.Context, cfg Config, forwardingEndpoint string) (Client, error) {
	options := mqtt.NewClientOptions()

	connectionString := fmt.Sprintf("ssl://%s:8883", cfg.host)
	options.AddBroker(connectionString)

	options.SetUsername(cfg.user)
	options.SetPassword(cfg.password)

	clientID := cfg.clientId
	if clientID == "" {
		clientID = "diwise/iot-agent/" + uuid.NewString()
	}
	options.SetClientID(clientID)

	options.SetDefaultPublishHandler(NewMessageHandler(ctx, forwardingEndpoint))

	options.SetKeepAlive(time.Duration(cfg.keepAlive) * time.Second)

	log := logging.GetFromContext(ctx).With(slog.String("mqtt-host", cfg.host))

	options.OnConnect = func(mc mqtt.Client) {
		log.Info("connected")
		for _, topic := range cfg.topics {
			log.Info("subscribing to topic", "topic", topic)
			token := mc.Subscribe(topic, 0, nil)
			token.Wait()
		}
	}

	options.OnConnectionLost = func(mc mqtt.Client, err error) {
		log.Error("connection lost", "err", err.Error())
		os.Exit(1)
	}

	options.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &mqttClient{
		cfg:     cfg,
		log:     log,
		options: options,
	}, nil
}

func NewConfigFromEnvironment(prefix string) (Config, error) {

	const topicEnvNamePattern string = "%sMQTT_TOPIC_%d"

	cfg := Config{
		enabled:   os.Getenv(fmt.Sprintf("%sMQTT_DISABLED", prefix)) != "true",
		host:      os.Getenv(fmt.Sprintf("%sMQTT_HOST", prefix)),
		keepAlive: 30,
		user:      os.Getenv(fmt.Sprintf("%sMQTT_USER", prefix)),
		password:  os.Getenv(fmt.Sprintf("%sMQTT_PASSWORD", prefix)),
		topics: []string{
			os.Getenv(fmt.Sprintf(topicEnvNamePattern, prefix, 0)),
		},
		clientId: os.Getenv(fmt.Sprintf("%sMQTT_CLIENT_ID", prefix)),
	}

	if !cfg.enabled {
		return cfg, nil
	}

	if cfg.host == "" {
		return cfg, fmt.Errorf("the mqtt host must be specified using the %sMQTT_HOST environment variable", prefix)
	}

	if cfg.topics[0] == "" {
		return cfg, fmt.Errorf("at least one topic (%sMQTT_TOPIC_0) must be added to the configuration", prefix)
	}

	customKeepAlive := os.Getenv(fmt.Sprintf("%sMQTT_KEEPALIVE", prefix))
	if customKeepAlive != "" {
		keepAlive, err := strconv.ParseInt(customKeepAlive, 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("custom keepalive value %s is not parseable to an int (%s)", customKeepAlive, err.Error())
		}
		cfg.keepAlive = keepAlive
	}

	const maxTopicCount int = 25

	for idx := 1; idx < maxTopicCount; idx++ {
		varName := fmt.Sprintf(topicEnvNamePattern, prefix, idx)
		value := os.Getenv(varName)

		if value != "" {
			cfg.topics = append(cfg.topics, value)
		}
	}

	return cfg, nil
}
