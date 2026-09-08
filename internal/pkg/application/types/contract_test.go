package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matryer/is"
)

// HARM-003: locks the device-status wire contract produced by the agent
// and consumed by iot-device-management and iot-events.
// Topic, content type and JSON field names must not change without a
// compatibility/migration plan and contract tests on the consumer side.
func TestStatusMessageWireContract(t *testing.T) {
	is := is.New(t)

	battery := 87.0
	m := &StatusMessage{
		DeviceID:     "internal-id-for-device",
		BatteryLevel: &battery,
		Tenant:       "default",
		Timestamp:    time.Date(2025, 4, 10, 11, 44, 1, 0, time.UTC),
	}

	is.Equal(m.TopicName(), "device-status")
	is.Equal(m.ContentType(), "application/json")

	var decoded map[string]any
	is.NoErr(json.Unmarshal(m.Body(), &decoded))

	is.Equal(decoded["deviceID"], "internal-id-for-device")
	is.Equal(decoded["tenant"], "default")
	is.Equal(decoded["batteryLevel"], 87.0)

	_, ok := decoded["timestamp"]
	is.True(ok)
}
