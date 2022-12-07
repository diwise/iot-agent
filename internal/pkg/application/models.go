package application

import (
	"encoding/json"
	"time"
)

type RXInfo struct {
	GatewayId string  `json:"gatewayId,omitempty"`
	UplinkId  string  `json:"uplinkId,omitempty"`
	Rssi      int32   `json:"rssi,omitempty"`
	Snr       float32 `json:"snr,omitempty"`
}

type TXInfo struct {
	Frequency uint32 `json:"frequency,omitempty"`
}

type Error struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

type SensorEvent struct {
	DevEui     string              `json:"devEui"`
	DeviceName string              `json:"deviceName"`
	SensorType string              `json:"sensorType"`
	FPort      uint8               `json:"fPort"`
	Data       []byte              `json:"data"`
	Object     json.RawMessage     `json:"object,omitempty"`
	Tags       map[string][]string `json:"tags,omitempty"`
	Timestamp  time.Time           `json:"timestamp"`
	RXInfo     RXInfo              `json:"rxInfo,omitempty"`
	TXInfo     TXInfo              `json:"txInfo,omitempty"`
	Error      Error               `json:"error,omitempty"`
}

func (s *SensorEvent) HasError() bool {
	return s.Error.Type != "" && s.Error.Message != ""
}