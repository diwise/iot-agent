package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/diwise/senml"
)

var ErrPayloadContainsNoData = errors.New("payload contains no data")

type RXInfo struct {
	GatewayId       string  `json:"gatewayId,omitempty"`
	UplinkId        string  `json:"uplinkId,omitempty"`
	Rssi            float64 `json:"rssi,omitempty"`
	Snr             float64 `json:"snr,omitempty"`
	SpreadingFactor float64 `json:"spreadingFactor,omitempty"`
	DataRate        int     `json:"dr"`
}

type TXInfo struct {
	Frequency int64 `json:"frequency,omitempty"`
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
	RXInfo     RXInfo              `json:"rxInfo"`
	TXInfo     TXInfo              `json:"txInfo"`
	Error      Error               `json:"error"`
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

func (m *StatusMessage) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}
