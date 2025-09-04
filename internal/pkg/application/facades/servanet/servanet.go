package servanet

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
	case "error":
		log.Debug("Handling error event")
		return handleErrorEvent(b)
	case "status":
		log.Debug("Handling status event")
		return handleStatusEvent(b)
	case "join":
		log.Debug("Handling join event (NOP)")
		return types.Event{}, fmt.Errorf("join events not supported")
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
		return types.Event{}, types.ErrDevEUIMissing
	}

	var data []byte
	data, err = base64.StdEncoding.DecodeString(uplinkEvent.Data)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(uplinkEvent.Data)
		if err != nil {
			return types.Event{}, err
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
			DR:        uplinkEvent.TxInfo.DR,
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

// UplinkEvent representerar hela meddelandet.
type UplinkEvent struct {
	ApplicationID     string            `json:"applicationID"`
	ApplicationName   string            `json:"applicationName"`
	DeviceName        string            `json:"deviceName"`
	DeviceProfileName string            `json:"deviceProfileName"`
	DeviceProfileID   string            `json:"deviceProfileID"`
	DevEUI            string            `json:"devEUI"`
	RxInfo            []RxInfo          `json:"rxInfo"`
	TxInfo            TxInfo            `json:"txInfo"`
	Adr               bool              `json:"adr"`
	FCnt              int               `json:"fCnt"`
	FPort             int               `json:"fPort"`
	Data              string            `json:"data"`
	Object            json.RawMessage   `json:"object"`
	ObjectJSON        json.RawMessage   `json:"objectJSON"`
	Tags              map[string]string `json:"tags"`
}

// RxInfo representerar mottagarinformation från en gateway.
type RxInfo struct {
	GatewayID string   `json:"gatewayID"`
	UplinkID  string   `json:"uplinkID"`
	Name      string   `json:"name"`
	RSSI      int      `json:"rssi"`
	LoRaSNR   float64  `json:"loRaSNR"`
	Location  Location `json:"location"`
}

// Location innehåller geografisk platsinformation.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// TxInfo representerar sändarinformation.
type TxInfo struct {
	Frequency int `json:"frequency"`
	DR        int `json:"dr"`
}

// ErrorEvent representerar ett felmeddelande från enheten, exempelvis vid uplink frame-counter retransmission.
type ErrorEvent struct {
	ApplicationID   string            `json:"applicationID"`
	ApplicationName string            `json:"applicationName"`
	DeviceName      string            `json:"deviceName"`
	DevEUI          string            `json:"devEUI"`
	Type            string            `json:"type"`
	Error           string            `json:"error"`
	FCnt            int               `json:"fCnt"`
	Tags            map[string]string `json:"tags"`
}

// StatusEvent representerar statusinformation från en enhet.
type StatusEvent struct {
	ApplicationID           string            `json:"applicationID"`
	ApplicationName         string            `json:"applicationName"`
	DeviceName              string            `json:"deviceName"`
	DevEUI                  string            `json:"devEUI"`
	Margin                  int               `json:"margin"`
	ExternalPowerSource     bool              `json:"externalPowerSource"`
	BatteryLevel            float64           `json:"batteryLevel"`
	BatteryLevelUnavailable bool              `json:"batteryLevelUnavailable"`
	Tags                    map[string]string `json:"tags"`
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
