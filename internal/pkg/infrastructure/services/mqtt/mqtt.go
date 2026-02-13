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
	Ready() bool
}

type Config struct {
	enabled   bool
	host      string
	port      int
	keepAlive int64
	user      string
	password  string
	topics    []string
	clientId  string
}

func NewClient(ctx context.Context, cfg Config, forwardingEndpoint string) (Client, error) {
	options := mqtt.NewClientOptions()

	connectionString := fmt.Sprintf("ssl://%s:%d", cfg.host, cfg.port)
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
	options.SetAutoReconnect(true)
	options.SetAutoAckDisabled(true)
	options.SetConnectRetry(false)
	options.SetMaxReconnectInterval(60 * time.Second)

	log := logging.GetFromContext(ctx).With(slog.String("mqtt-host", cfg.host), slog.Int("mqtt-port", cfg.port))

	options.OnConnect = func(mc mqtt.Client) {
		log.Info("connected")
		for _, topic := range cfg.topics {
			log.Info("subscribing to topic", "topic", topic)
			token := mc.Subscribe(topic, 1, nil)
			token.Wait()
			if token.Error() != nil {
				log.Error("subscribe failed", "topic", topic, "err", token.Error())
			}
		}
	}

	options.OnConnectionLost = func(mc mqtt.Client, err error) {
		if err != nil {
			log.Error("connection lost", "err", err)
			return
		}

		log.Warn("connection lost")
	}

	options.OnReconnecting = func(mc mqtt.Client, co *mqtt.ClientOptions) {
		log.Warn("attempting mqtt reconnect")
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

	//TODO: clientID should be <username>-<uuid>

	cfg := Config{
		enabled:   os.Getenv(fmt.Sprintf("%sMQTT_DISABLED", prefix)) != "true",
		host:      os.Getenv(fmt.Sprintf("%sMQTT_HOST", prefix)),
		port:      8883,
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

	customPort := os.Getenv(fmt.Sprintf("%sMQTT_PORT", prefix))
	if customPort != "" {
		port, err := strconv.Atoi(customPort)
		if err != nil {
			return cfg, fmt.Errorf("custom port value %s is not parseable to an int (%s)", customPort, err.Error())
		}
		if port < 1 || port > 65535 {
			return cfg, fmt.Errorf("custom port value %s is outside valid range 1-65535", customPort)
		}
		cfg.port = port
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
