package mqtt

import (
	"crypto/tls"
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Client interface {
	Start() error
	Stop()
}

type Config struct {
	host     string
	user     string
	password string
	topics   []string
}

func NewClient(logger zerolog.Logger, cfg Config) (Client, error) {
	options := mqtt.NewClientOptions()

	connectionString := fmt.Sprintf("tls://%s:8883", cfg.host)
	options.AddBroker(connectionString)

	options.Username = cfg.user
	options.Password = cfg.password

	options.SetClientID("diwise/iot-agent/" + uuid.NewString())
	options.SetDefaultPublishHandler(NewMessageHandler(logger))

	options.OnConnect = func(mc mqtt.Client) {
		logger.Info().Msg("connected")
		for _, topic := range cfg.topics {
			logger.Info().Msgf("subscribing to %s", topic)
			mc.Subscribe(topic, 0, nil)
		}
	}

	options.OnConnectionLost = func(mc mqtt.Client, err error) {
		logger.Fatal().Err(err).Msg("connection lost")
	}

	options.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &mqttClient{
		cfg:     cfg,
		log:     logger.With().Str("mqtt-host", cfg.host).Logger(),
		options: options,
	}, nil
}

func NewConfigFromEnvironment() (Config, error) {
	cfg := Config{
		host:     os.Getenv("MQTT_HOST"),
		user:     os.Getenv("MQTT_USER"),
		password: os.Getenv("MQTT_PASSWORD"),
		topics:   []string{"application/8/device/#", "application/24/device/#", "application/53/device/#"},
	}

	return cfg, nil
}
