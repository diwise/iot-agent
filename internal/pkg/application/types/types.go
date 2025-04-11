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
var ErrUnknownMessageType = errors.New("unknown message type")

type Event struct {
	DevEUI     string   `json:"devEUI"`
	Name       string   `json:"name,omitempty"`
	SensorType string   `json:"sensorType,omitempty"`
	Source     string   `json:"source,omitempty"`
	Location   Location `json:"location"`

	RX *RX `json:"rx,omitempty"`
	TX *TX `json:"tx,omitempty"`

	FCnt int `json:"fCnt"`

	Payload *Payload `json:"payload,omitempty"`
	Status  *Status  `json:"status,omitempty"`
	Error   *Error   `json:"error,omitempty"`

	Tags      map[string][]string `json:"tags,omitempty"`
	Timestamp time.Time           `json:"timestamp"`
}

type Payload struct {
	FPort  int             `json:"fPort"`
	Data   []byte          `json:"data"`
	Object json.RawMessage `json:"object,omitempty"`
}

type TX struct {
	Frequency       int64   `json:"frequency"`
	SpreadingFactor float64 `json:"spreadingFactor"`
	DR              int     `json:"dr"`
}

type RX struct {
	RSSI    float64 `json:"rssi"`
	LoRaSNR float64 `json:"loRaSNR"`
}

type Status struct {
	Margin                  int     `json:"margin"`
	BatteryLevelUnavailable bool    `json:"batteryLevelUnavailable"`
	BatteryLevel            float64 `json:"batteryLevel"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

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
	DeviceID string `json:"deviceID"`

	BatteryLevel int `json:"batteryLevel"`

	Code     string   `json:"statusCode"`
	Messages []string `json:"statusMessages,omitempty"`

	RSSI            float64 `json:"rssi"`
	LoRaSNR         float64 `json:"loRaSNR"`
	Frequency       int64   `json:"frequency"`
	SpreadingFactor float64 `json:"spreadingFactor"`
	DR              int     `json:"dr"`

	Tenant    string    `json:"tenant"`
	Timestamp time.Time `json:"timestamp"`
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
