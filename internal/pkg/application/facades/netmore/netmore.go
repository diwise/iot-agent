package netmore

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	. "github.com/diwise/iot-agent/internal/pkg/application/types"
)

func HandleUplinkEvent(b []byte) (SensorEvent, error) {
	var uplinkEvents []UplinkEvent
	err := json.Unmarshal(b, &uplinkEvents)
	if err != nil {
		return SensorEvent{}, err
	}

	if len(uplinkEvents) == 0 {
		return SensorEvent{}, ErrPayloadContainsNoData
	}

	uplinkEvent := uplinkEvents[0]

	var data []byte
	data, err = hex.DecodeString(uplinkEvent.Payload)

	if err != nil {
		return SensorEvent{}, err
	}

	ue := SensorEvent{
		DevEui:     uplinkEvent.DevEui,
		DeviceName: uplinkEvent.SensorType, // no DeviceName is received from Netmore
		SensorType: uplinkEvent.SensorType,
		FPort:      atoi[uint8](uplinkEvent.FPort),
		Data:       data,
		RXInfo: RXInfo{
			GatewayId:       uplinkEvent.GatewayIdentifier,
			Rssi:            atof(uplinkEvent.RSSI),
			Snr:             atof(uplinkEvent.SNR),
			SpreadingFactor: atof(uplinkEvent.SpreadingFactor),
			DataRate:        uplinkEvent.DR,
		},
		TXInfo: TXInfo{
			Frequency: uplinkEvent.Freq,
		},
		Tags:      uplinkEvent.Tags,
		Timestamp: uplinkEvent.Timestamp,
	}

	return ue, nil
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
