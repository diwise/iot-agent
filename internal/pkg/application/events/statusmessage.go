package events

import (
	"time"
)

type StatusMessage struct {
	DeviceID     string   `json:"deviceID"`
	BatteryLevel int      `json:"batteryLevel"`
	Code         int      `json:"statusCode"`
	Messages     []string `json:"statusMessages,omitempty"`
	Tenant       string   `json:"tenant,omitempty"`
	Timestamp    string   `json:"timestamp"`
}

func NewStatusMessage(deviceID string, options ...func(*StatusMessage)) *StatusMessage {
	msg := &StatusMessage{
		DeviceID:  deviceID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	for _, op := range options {
		op(msg)
	}

	return msg
}

func WithStatus(code int, messages []string) func(*StatusMessage) {
	return func(sm *StatusMessage) {
		sm.Code = code
		sm.Messages = messages
	}
}

func WithTenant(tenant string) func(*StatusMessage) {
	return func(sm *StatusMessage) {
		sm.Tenant = tenant		
	}
}

func WithBatteryLevel(batteryLevel int) func(*StatusMessage) {
	return func(sm *StatusMessage) {
		sm.BatteryLevel = batteryLevel
	}
}

func (m *StatusMessage) ContentType() string {
	return "application/json"
}

func (m *StatusMessage) TopicName() string {
	return "device-status"
}
