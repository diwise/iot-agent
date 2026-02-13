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
