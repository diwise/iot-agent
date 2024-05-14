package application

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/diwise/senml"
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

type Measurement struct {
	Timestamp time.Time  `json:"timestamp"`
	Pack      senml.Pack `json:"pack"`
}

type DecoderErr struct {
	Code      int
	Messages  []string
	Timestamp time.Time
}

func (e *DecoderErr) Error() string {
	return fmt.Sprintf("error code %d with messages: %s", e.Code, strings.Join(e.Messages, ", "))
}
