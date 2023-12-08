package events

import "time"

type StatusMessage struct {
	DeviceID     string    `json:"deviceID"`
	BatteryLevel int       `json:"batteryLevel"`
	Code         int       `json:"statusCode"`
	Messages     []string  `json:"statusMessages,omitempty"`
	Tenant       string    `json:"tenant"`
	Timestamp    time.Time `json:"timestamp"`
}

func (m *StatusMessage) ContentType() string {
	return "application/json"
}

func (m *StatusMessage) TopicName() string {
	return "device-status"
}
