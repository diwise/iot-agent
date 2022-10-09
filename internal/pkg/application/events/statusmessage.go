package events

import (
	"strings"
	"time"
)

type StatusMessage struct {
	DeviceID     string  `json:"deviceID"`
	BatteryLevel int     `json:"batteryLevel"`
	Error        *string `json:"error,omitempty"`
	Status       Status  `json:"status"`
	Timestamp    string  `json:"timestamp"`
}

type Status struct {
	Code     int      `json:"statusCode"`
	Messages []string `json:"statusMessages,omitempty"`
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
		sm.Status = Status{
			Code:     code,
			Messages: messages,
		}
	}
}

func WithBatteryLevel(batteryLevel int) func(*StatusMessage) {
	return func(sm *StatusMessage) {
		sm.BatteryLevel = batteryLevel
	}
}

func WithError(err string) func(*StatusMessage) {
	return func(sm *StatusMessage) {
		if strings.Trim(err, " ") == "" {
			sm.Error = nil
		} else {
			sm.Error = &err
		}
	}
}

func (m *StatusMessage) ContentType() string {
	return "application/json"
}

func (m *StatusMessage) TopicName() string {
	return "device-status"
}
