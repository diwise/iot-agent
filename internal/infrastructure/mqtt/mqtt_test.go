package mqtt

import "testing"

func TestNewConfigFromEnvironmentUsesDefaultPort(t *testing.T) {
	t.Setenv("MQTT_DISABLED", "false")
	t.Setenv("MQTT_HOST", "broker.example")
	t.Setenv("MQTT_TOPIC_0", "foo/#")
	t.Setenv("MQTT_PORT", "")

	cfg, err := NewConfigFromEnvironment("")
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}

	if cfg.port != 8883 {
		t.Fatalf("expected default port 8883, got %d", cfg.port)
	}

	if cfg.session != sessionModeEphemeral {
		t.Fatalf("expected default session mode %q, got %q", sessionModeEphemeral, cfg.session)
	}
}

func TestNewConfigFromEnvironmentUsesCustomPort(t *testing.T) {
	t.Setenv("MQTT_DISABLED", "false")
	t.Setenv("MQTT_HOST", "broker.example")
	t.Setenv("MQTT_TOPIC_0", "foo/#")
	t.Setenv("MQTT_PORT", "1884")

	cfg, err := NewConfigFromEnvironment("")
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}

	if cfg.port != 1884 {
		t.Fatalf("expected custom port 1884, got %d", cfg.port)
	}
}

func TestNewConfigFromEnvironmentRejectsInvalidPort(t *testing.T) {
	t.Setenv("MQTT_DISABLED", "false")
	t.Setenv("MQTT_HOST", "broker.example")
	t.Setenv("MQTT_TOPIC_0", "foo/#")
	t.Setenv("MQTT_PORT", "abc")

	_, err := NewConfigFromEnvironment("")
	if err == nil {
		t.Fatal("expected error for invalid mqtt port")
	}
}

func TestNewConfigFromEnvironmentUsesDurableSession(t *testing.T) {
	t.Setenv("MQTT_DISABLED", "false")
	t.Setenv("MQTT_HOST", "broker.example")
	t.Setenv("MQTT_TOPIC_0", "foo/#")
	t.Setenv("MQTT_SESSION_MODE", "durable")
	t.Setenv("MQTT_CLIENT_ID", "iot-agent-durable")

	cfg, err := NewConfigFromEnvironment("")
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}

	if cfg.session != sessionModeDurable {
		t.Fatalf("expected durable session mode, got %q", cfg.session)
	}

	if cfg.clientId != "iot-agent-durable" {
		t.Fatalf("expected durable client id to be preserved, got %q", cfg.clientId)
	}
}

func TestNewConfigFromEnvironmentRejectsInvalidSessionMode(t *testing.T) {
	t.Setenv("MQTT_DISABLED", "false")
	t.Setenv("MQTT_HOST", "broker.example")
	t.Setenv("MQTT_TOPIC_0", "foo/#")
	t.Setenv("MQTT_SESSION_MODE", "sticky")

	_, err := NewConfigFromEnvironment("")
	if err == nil {
		t.Fatal("expected error for invalid mqtt session mode")
	}
}

func TestNewConfigFromEnvironmentRequiresClientIDForDurableSession(t *testing.T) {
	t.Setenv("MQTT_DISABLED", "false")
	t.Setenv("MQTT_HOST", "broker.example")
	t.Setenv("MQTT_TOPIC_0", "foo/#")
	t.Setenv("MQTT_SESSION_MODE", "durable")

	_, err := NewConfigFromEnvironment("")
	if err == nil {
		t.Fatal("expected error when durable mqtt session lacks client id")
	}
}
