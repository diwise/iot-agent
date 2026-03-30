package mqtt

import (
	"context"
	"strings"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestNewClientUsesEphemeralSessionDefaults(t *testing.T) {
	client, err := NewClient(context.Background(), Config{
		enabled:   true,
		host:      "broker.example",
		port:      8883,
		keepAlive: 30,
		topics:    []string{"foo/#"},
		session:   sessionModeEphemeral,
	}, "http://example.invalid/api/v0/messages")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	mqttClient, ok := client.(*mqttClient)
	if !ok {
		t.Fatalf("unexpected client type %T", client)
	}

	if !mqttClient.options.CleanSession {
		t.Fatal("expected clean session to be enabled for ephemeral mqtt sessions")
	}

	if mqttClient.options.ResumeSubs {
		t.Fatal("expected resume subscriptions to be disabled for ephemeral mqtt sessions")
	}

	if !strings.HasPrefix(mqttClient.options.ClientID, "diwise/iot-agent/") {
		t.Fatalf("expected random ephemeral client id, got %q", mqttClient.options.ClientID)
	}
}

func TestNewClientUsesDurableSessionOptions(t *testing.T) {
	client, err := NewClient(context.Background(), Config{
		enabled:   true,
		host:      "broker.example",
		port:      8883,
		keepAlive: 30,
		topics:    []string{"foo/#"},
		clientId:  "iot-agent-durable",
		session:   sessionModeDurable,
	}, "http://example.invalid/api/v0/messages")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	mqttClient, ok := client.(*mqttClient)
	if !ok {
		t.Fatalf("unexpected client type %T", client)
	}

	if mqttClient.options.CleanSession {
		t.Fatal("expected clean session to be disabled for durable mqtt sessions")
	}

	if !mqttClient.options.ResumeSubs {
		t.Fatal("expected resume subscriptions to be enabled for durable mqtt sessions")
	}

	if mqttClient.options.Order {
		t.Fatal("expected ordered callback processing to be disabled")
	}

	if mqttClient.options.ClientID != "iot-agent-durable" {
		t.Fatalf("expected configured durable client id, got %q", mqttClient.options.ClientID)
	}
}

func TestReadyReturnsFalseWithoutUnderlyingClient(t *testing.T) {
	client := &mqttClient{}
	if client.Ready() {
		t.Fatal("expected mqtt client without underlying client to report not ready")
	}
}

func TestReadyReflectsUnderlyingConnectionState(t *testing.T) {
	client := &mqttClient{client: &fakePahoClient{connectionOpen: true}}
	if !client.Ready() {
		t.Fatal("expected mqtt client with open connection to report ready")
	}
}

func TestStopDisconnectsClientAndClearsReference(t *testing.T) {
	fakeClient := &fakePahoClient{}
	client := &mqttClient{client: fakeClient}
	client.running.Store(true)

	client.Stop()

	if client.running.Load() {
		t.Fatal("expected Stop to mark mqtt client as not running")
	}

	if fakeClient.disconnectCalls != 1 {
		t.Fatalf("expected Stop to disconnect underlying client once, got %d", fakeClient.disconnectCalls)
	}

	if client.client != nil {
		t.Fatal("expected Stop to clear the underlying client reference")
	}

	if client.Ready() {
		t.Fatal("expected mqtt client to report not ready after Stop")
	}
}

func TestNewClientRejectsDurableSessionWithoutClientID(t *testing.T) {
	_, err := NewClient(context.Background(), Config{
		enabled:   true,
		host:      "broker.example",
		port:      8883,
		keepAlive: 30,
		topics:    []string{"foo/#"},
		session:   sessionModeDurable,
	}, "http://example.invalid/api/v0/messages")
	if err == nil {
		t.Fatal("expected error when durable mqtt session lacks client id")
	}
}

type fakePahoClient struct {
	connectionOpen  bool
	disconnectCalls int
}

func (f *fakePahoClient) IsConnected() bool {
	return f.connectionOpen
}

func (f *fakePahoClient) IsConnectionOpen() bool {
	return f.connectionOpen
}

func (f *fakePahoClient) Connect() mqtt.Token {
	return nil
}

func (f *fakePahoClient) Disconnect(quiesce uint) {
	f.disconnectCalls++
	f.connectionOpen = false
}

func (f *fakePahoClient) Publish(topic string, qos byte, retained bool, payload any) mqtt.Token {
	return nil
}

func (f *fakePahoClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return nil
}

func (f *fakePahoClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	return nil
}

func (f *fakePahoClient) Unsubscribe(topics ...string) mqtt.Token {
	return nil
}

func (f *fakePahoClient) AddRoute(topic string, callback mqtt.MessageHandler) {}

func (f *fakePahoClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}
