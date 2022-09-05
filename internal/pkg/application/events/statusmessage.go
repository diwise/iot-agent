package events

import "time"

type StatusMessage struct {
	DeviceID   string `json:"deviceID"`
	StatusCode int    `json:"statusCode"`
	Timestamp  string `json:"timestamp"`
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

func StatusCode(statusCode int) func(*StatusMessage) {
	return func(sm *StatusMessage) {
		sm.StatusCode = statusCode
	}
}

func (m *StatusMessage) ContentType() string {
	return "application/json"
}

func (m *StatusMessage) TopicName() string {
	return "device-status"
}
