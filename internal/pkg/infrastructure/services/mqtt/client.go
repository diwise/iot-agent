package mqtt

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttClient struct {
	cfg      Config
	log      *slog.Logger
	options  *mqtt.ClientOptions
	running  atomic.Bool
	client   mqtt.Client
	clientMu sync.RWMutex
}

const (
	startupConnectAttempts  = 5
	startupBackoffBase      = 10 * time.Second
	startupBackoffMax       = 60 * time.Second
	disconnectExitThreshold = 3 * time.Minute
	connectionMonitorTick   = 1 * time.Second
)

func (c *mqttClient) Start() error {

	if !c.cfg.enabled {
		c.log.Warn("mqtt has been explicitly disabled with MQTT_DISABLED=true and will therefore not start")
		return nil
	}

	if !c.running.CompareAndSwap(false, true) {
		c.log.Warn("mqtt client is already running")
		return nil
	}

	go c.run()

	return nil
}

func (c *mqttClient) run() {
	defer c.running.Store(false)

	client := mqtt.NewClient(c.options)

	c.clientMu.Lock()
	c.client = client
	c.clientMu.Unlock()

	connected := false

	for attempt := 1; attempt <= startupConnectAttempts && c.running.Load(); attempt++ {
		token := client.Connect()
		token.Wait()
		if token.Error() == nil {
			connected = true
			break
		}

		c.log.Error("connection error", "attempt", attempt, "max_attempts", startupConnectAttempts, "err", token.Error())

		if attempt == startupConnectAttempts {
			break
		}

		backoff := startupBackoff(attempt)
		c.log.Warn("retrying mqtt connect", "backoff", backoff)
		time.Sleep(backoff)
	}

	if !connected && c.running.Load() {
		c.log.Error("failed to establish mqtt connection after retries, exiting")
		os.Exit(1)
		return
	}

	var disconnectedAt time.Time

	for c.running.Load() {
		if client.IsConnectionOpen() {
			disconnectedAt = time.Time{}
		} else {
			if disconnectedAt.IsZero() {
				disconnectedAt = time.Now()
				c.log.Warn("mqtt disconnected, waiting for reconnect")
			}

			disconnectedFor := time.Since(disconnectedAt)
			if disconnectedFor >= disconnectExitThreshold {
				c.log.Error("mqtt disconnected for too long, exiting", "disconnected_for", disconnectedFor, "threshold", disconnectExitThreshold)
				os.Exit(1)
				return
			}
		}

		time.Sleep(connectionMonitorTick)
	}

	if client.IsConnectionOpen() {
		client.Disconnect(250)
	}
}

func startupBackoff(attempt int) time.Duration {
	if attempt < 1 {
		return startupBackoffBase
	}

	backoff := startupBackoffBase << (attempt - 1)
	if backoff > startupBackoffMax {
		return startupBackoffMax
	}

	return backoff
}

func (c *mqttClient) Ready() bool {
	c.clientMu.RLock()
	defer c.clientMu.RUnlock()

	return c.client != nil && c.client.IsConnectionOpen()
}

func (c *mqttClient) Stop() {
	c.running.Store(false)

	c.clientMu.RLock()
	defer c.clientMu.RUnlock()

	if c.client != nil && c.client.IsConnectionOpen() {
		c.client.Disconnect(250)
	}
}
