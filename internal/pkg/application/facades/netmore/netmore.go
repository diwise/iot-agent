package netmore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
)

func HandleEvent(ctx context.Context, messageType string, b []byte) (types.Event, error) {
	if messageType == "" {
		messageType = "payload"
	}

	if messageType != "payload" {
		return types.Event{}, types.ErrUnknownMessageType
	}

	var uplinkEvents []UplinkEvent
	err := json.Unmarshal(b, &uplinkEvents)
	if err != nil {
		return types.Event{}, err
	}

	if len(uplinkEvents) == 0 {
		return types.Event{}, types.ErrPayloadContainsNoData
	}

	uplinkEvent := uplinkEvents[0]

	if uplinkEvent.DevEui == "" {
		return types.Event{}, types.ErrDevEUIMissing
	}

	var data []byte
	data, err = hex.DecodeString(uplinkEvent.Payload)
	if err != nil {
		return types.Event{}, err
	}

	e := types.Event{
		DevEUI:     uplinkEvent.DevEui,
		SensorType: uplinkEvent.SensorType,
		FCnt:       uplinkEvent.FCntUp,
		Location: types.Location{
			Latitude:  uplinkEvent.Latitude,
			Longitude: uplinkEvent.Longitude,
		},
		Payload: &types.Payload{
			FPort: atoi[int](uplinkEvent.FPort),
			Data:  data,
		},
		RX: &types.RX{
			RSSI:    atof(uplinkEvent.RSSI),
			LoRaSNR: atof(uplinkEvent.SNR),
		},
		TX: &types.TX{
			Frequency:       uplinkEvent.Freq,
			SpreadingFactor: atof(uplinkEvent.SpreadingFactor),
			DR:              uplinkEvent.DR,
		},
		Tags:      uplinkEvent.Tags,
		Timestamp: uplinkEvent.Timestamp,
	}

	return e, nil
}

type UplinkEvent struct {
	DevEui            string              `json:"devEui"`
	SensorType        string              `json:"sensorType"`
	MessageType       string              `json:"messageType"`
	Timestamp         time.Time           `json:"timestamp"`
	Payload           string              `json:"payload"`
	FCntUp            int                 `json:"fCntUp"`
	ToA               *float64            `json:"toa"` // Används pekare då värdet kan vara null
	Freq              int64               `json:"freq"`
	BatteryLevel      string              `json:"batteryLevel"`
	Ack               bool                `json:"ack"`
	SpreadingFactor   string              `json:"spreadingFactor"`
	DR                int                 `json:"dr"`
	RSSI              string              `json:"rssi"`
	SNR               string              `json:"snr"`
	GatewayIdentifier string              `json:"gatewayIdentifier"`
	FPort             string              `json:"fPort"`
	Latitude          float64             `json:"latitude"`
	Longitude         float64             `json:"longitude"`
	Tags              map[string][]string `json:"tags"`
	Gateways          []Gateway           `json:"gateways"`
}

type Gateway struct {
	RSSI              string `json:"rssi"`
	SNR               string `json:"snr"`
	GatewayIdentifier string `json:"gatewayIdentifier"`
	GwEui             string `json:"gwEui"`
	MAC               string `json:"mac"`
	Antenna           int    `json:"antenna"`
}

func atoi[T int | int32 | uint8 | uint32](s string) T {
	if i, err := strconv.Atoi(s); err == nil {
		return T(i)
	}
	return 0
}

func atof[T float64](s string) T {
	if i, err := strconv.ParseFloat(s, 64); err == nil {
		return T(i)
	}
	return 0.0
}
