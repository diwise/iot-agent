package chirpstackv4

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

func HandleEvent(ctx context.Context, messageType string, b []byte) (types.Event, error) {
	log := logging.GetFromContext(ctx)

	switch messageType {
	case "up":
		log.Debug("Handling uplink event")
		evt, err := handleUplinkEvent(b)
		if err != nil {
			log.Error("Failed to handle uplink event", "err", err)
		}
		return evt, err
	case "status":
		log.Debug("Handling status event")
		return handleStatusEvent(b)
	case "error":
		log.Debug("Handling error event")
		return handleErrorEvent(b)
	case "txack", "down", "join":
		log.Debug(fmt.Sprintf("Handling %s event (NOP)", messageType))
		return types.Event{}, nil
	default:
		log.Debug("Handling unknown event type as uplink event", "type", messageType)
		return handleUplinkEvent(b)
	}
}

func handleErrorEvent(b []byte) (types.Event, error) {
	var errorEvent ErrorEvent
	err := json.Unmarshal(b, &errorEvent)
	if err != nil {
		return types.Event{}, err
	}

	e := types.Event{
		DevEUI:    errorEvent.DevEUI,
		Name:      errorEvent.DeviceName,
		Tags:      mapToMapArr(errorEvent.Tags),
		Timestamp: time.Now().UTC(),
		Error: &types.Error{
			Type:    errorEvent.Type,
			Message: errorEvent.Error,
		},
	}

	return e, nil
}

func handleStatusEvent(b []byte) (types.Event, error) {
	var statusEvent StatusEvent
	err := json.Unmarshal(b, &statusEvent)
	if err != nil {
		return types.Event{}, err
	}

	e := types.Event{
		DevEUI:    statusEvent.DevEUI,
		Name:      statusEvent.DeviceName,
		Tags:      mapToMapArr(statusEvent.Tags),
		Timestamp: time.Now().UTC(),
		Status: &types.Status{
			Margin:                  statusEvent.Margin,
			BatteryLevel:            statusEvent.BatteryLevel,
			BatteryLevelUnavailable: statusEvent.BatteryLevelUnavailable,
		},
	}

	return e, nil
}

func handleUplinkEvent(b []byte) (types.Event, error) {
	var uplinkEvent UplinkEvent
	err := json.Unmarshal(b, &uplinkEvent)
	if err != nil {
		return types.Event{}, err
	}

	if uplinkEvent.DevEUI == "" {
		return types.Event{}, types.ErrSensorIDMissing
	}

	var data []byte
	if uplinkEvent.Data != "" {
		data, err = base64.StdEncoding.DecodeString(uplinkEvent.Data)
		if err != nil {
			data, err = base64.RawStdEncoding.DecodeString(uplinkEvent.Data)
			if err != nil {
				return types.Event{}, err
			}
		}
	}

	if uplinkEvent.Data == "" && uplinkEvent.Object == nil {
		return types.Event{}, types.ErrPayloadContainsNoData
	}

	var objectJSON json.RawMessage
	if uplinkEvent.Object != nil {
		objectJSON = uplinkEvent.Object
	} else {
		objectJSON = uplinkEvent.ObjectJSON
	}

	e := types.Event{
		DevEUI:     uplinkEvent.DevEUI,
		Name:       uplinkEvent.DeviceName,
		SensorType: uplinkEvent.DeviceProfileName,
		FCnt:       uplinkEvent.FCnt,
		Tags:       mapToMapArr(uplinkEvent.Tags),
		Timestamp:  time.Now().UTC(),

		Payload: &types.Payload{
			FPort:  uplinkEvent.FPort,
			Data:   data,
			Object: objectJSON,
		},

		TX: &types.TX{
			Frequency: int64(uplinkEvent.TxInfo.Frequency),
			DR:        uplinkEvent.Dr,
		},
	}

	if len(uplinkEvent.RxInfo) > 0 {
		e.Location = types.Location{
			Latitude:  uplinkEvent.RxInfo[0].Location.Latitude,
			Longitude: uplinkEvent.RxInfo[0].Location.Longitude,
		}

		e.RX = &types.RX{
			RSSI:    float64(uplinkEvent.RxInfo[0].RSSI),
			LoRaSNR: uplinkEvent.RxInfo[0].LoRaSNR,
		}
	}

	return e, nil
}

// UplinkEvent representerar ett fullständigt uplink-meddelande från ChirpStack v4.
type UplinkEvent struct {
	// Grundinformation
	ApplicationID   string `json:"applicationID"`
	ApplicationName string `json:"applicationName"`
	DeviceName      string `json:"deviceName"`
	DevEUI          string `json:"devEUI"`

	// Mottagningsinformation
	RxInfo []RxInfo `json:"rxInfo"`

	// Sändningsinformation
	TxInfo TxInfo `json:"txInfo"`

	// Nätverksegenskaper
	Adr        bool              `json:"adr"`
	Dr         int               `json:"dr"`
	FCnt       int               `json:"fCnt"`
	FPort      int               `json:"fPort"`
	Data       string            `json:"data"`   // Base64‑kodat
	Object     json.RawMessage   `json:"object"` // JSON-objekt
	ObjectJSON json.RawMessage   `json:"objectJSON"`
	Tags       map[string]string `json:"tags"`

	// Ytterligare fält
	ConfirmedUplink   bool      `json:"confirmed_uplink"`
	DevAddr           string    `json:"dev_addr"`
	PublishedAt       time.Time `json:"published_at"`
	DeviceProfileID   string    `json:"deviceProfileID"`
	DeviceProfileName string    `json:"deviceProfileName"`
}

// RxInfo representerar ett mottaget RX-objekt (gateway-specifik metadata).
type RxInfo struct {
	GatewayID         string    `json:"gatewayID"`
	Time              time.Time `json:"time"`
	TimeSinceGPSEpoch *int64    `json:"timeSinceGPSEpoch"`
	RSSI              int       `json:"rssi"`
	LoRaSNR           float64   `json:"loRaSNR"`
	Channel           int       `json:"channel"`
	RFChain           int       `json:"rfChain"`
	Board             int       `json:"board"`
	Antenna           int       `json:"antenna"`
	Location          Location  `json:"location"`
	FineTimestampType string    `json:"fineTimestampType"`
	Context           string    `json:"context"`
	UplinkID          string    `json:"uplinkID"`
}

// Location representerar geografisk plats med latitude, longitude och altitude.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// TxInfo representerar sändningsinformation från enheten.
type TxInfo struct {
	Frequency          int                `json:"frequency"`
	Modulation         string             `json:"modulation"`
	LoRaModulationInfo LoRaModulationInfo `json:"loRaModulationInfo"`
}

// LoRaModulationInfo innehåller specifika inställningar för LoRa-moduleringen.
type LoRaModulationInfo struct {
	Bandwidth             int    `json:"bandwidth"`
	SpreadingFactor       int    `json:"spreadingFactor"`
	CodeRate              string `json:"codeRate"`
	PolarizationInversion bool   `json:"polarizationInversion"`
}

// StatusEvent representerar statusinformationen för en enhet.
type StatusEvent struct {
	ApplicationID           string            `json:"applicationID"`
	ApplicationName         string            `json:"applicationName"`
	DeviceName              string            `json:"deviceName"`
	DevEUI                  string            `json:"devEUI"`
	Margin                  int               `json:"margin"`
	ExternalPowerSource     bool              `json:"externalPowerSource"`
	BatteryLevelUnavailable bool              `json:"batteryLevelUnavailable"`
	BatteryLevel            float64           `json:"batteryLevel"`
	Tags                    map[string]string `json:"tags"`
}

// ErrorEvent representerar information om ett error-meddelande.
type ErrorEvent struct {
	ApplicationID   string            `json:"applicationID"`
	ApplicationName string            `json:"applicationName"`
	DeviceName      string            `json:"deviceName"`
	DevEUI          string            `json:"devEUI"`
	Type            string            `json:"type"`
	Error           string            `json:"error"`
	Tags            map[string]string `json:"tags"`
}

func mapToMapArr(m map[string]string) map[string][]string {
	ma := make(map[string][]string, 0)

	if m == nil {
		return ma
	}

	for k, v := range m {
		ma[k] = []string{v}
	}

	return ma
}
