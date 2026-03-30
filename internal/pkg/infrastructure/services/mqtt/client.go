package mqtt

import (
	"log/slog"
	"sync"
	"sync/atomic"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttClient struct {
	cfg       Config
	log       *slog.Logger
	options   *mqtt.ClientOptions
	running   atomic.Bool
	client    mqtt.Client
	forwarder *messageForwarder
	clientMu  sync.RWMutex
}

func (c *mqttClient) Start() error {

	if !c.cfg.enabled {
		c.log.Warn("mqtt has been explicitly disabled with MQTT_DISABLED=true and will therefore not start")
		return nil
	}

	if !c.running.CompareAndSwap(false, true) {
		c.log.Warn("mqtt client is already running")
		return nil
	}

	client := mqtt.NewClient(c.options)

	c.clientMu.Lock()
	c.client = client
	c.clientMu.Unlock()

	token := client.Connect()
	go func() {
		token.Wait()
		if err := token.Error(); err != nil && c.running.Load() {
			c.log.Error("mqtt connect failed", "err", err)
		}
	}()

	return nil
}

func (c *mqttClient) Ready() bool {
	c.clientMu.RLock()
	defer c.clientMu.RUnlock()

	return c.client != nil && c.client.IsConnectionOpen()
}

func (c *mqttClient) Stop() {
	c.running.Store(false)

	c.clientMu.Lock()
	client := c.client
	c.client = nil
	forwarder := c.forwarder
	c.forwarder = nil
	c.clientMu.Unlock()

	if client != nil {
		client.Disconnect(250)
	}

	if forwarder != nil {
		forwarder.Close()
	}
}
