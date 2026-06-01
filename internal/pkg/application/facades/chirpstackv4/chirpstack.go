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
	case "up", "uplink":
		log.Debug("Handling uplink event")
		evt, err := handleUplinkEvent(b)
		if err != nil {
			log.Error("Failed to handle uplink event", "err", err)
		}
		return evt, err
	case "status":
		log.Debug("Handling status event")
		return handleStatusEvent(b)
	case "log", "error":
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
	var errorEvent LogEvent
	err := json.Unmarshal(b, &errorEvent)
	if err != nil {
		return types.Event{}, err
	}

	e := types.Event{
		DevEUI:     errorEvent.DeviceInfo.DevEUI,
		Name:       errorEvent.DeviceInfo.DeviceName,
		Tags:       mapToMapArr(errorEvent.DeviceInfo.Tags),
		Timestamp:  time.Now().UTC(),
		Error: &types.Error{
			Level:   errorEvent.Level,
			Type:    errorEvent.Code,
			Message: errorEvent.Description,
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
		DevEUI:     statusEvent.DeviceInfo.DevEUI,
		Name:       statusEvent.DeviceInfo.DeviceName,
		SensorType: statusEvent.DeviceInfo.DeviceProfileName,		
		Tags:       mapToMapArr(statusEvent.DeviceInfo.Tags),
		Timestamp:  statusEvent.Time.UTC(),
		Status: &types.Status{
			Margin:                  statusEvent.Margin,
			BatteryLevel:            statusEvent.BatteryLevel,
			BatteryLevelUnavailable: statusEvent.BatteryLevel == 0.0,
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

	if uplinkEvent.DeviceInfo.DevEUI == "" {
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
		DevEUI:     uplinkEvent.DeviceInfo.DevEUI,
		Name:       uplinkEvent.DeviceInfo.DeviceName,
		SensorType: uplinkEvent.DeviceInfo.DeviceProfileName,
		FCnt:       int(uplinkEvent.FCnt),
		Tags:       mapToMapArr(uplinkEvent.DeviceInfo.Tags),
		Timestamp:  uplinkEvent.Time.UTC(),

		Payload: &types.Payload{
			FPort:  uplinkEvent.FPort,
			Data:   data,
			Object: objectJSON,
		},

		TX: &types.TX{
			Frequency: int64(uplinkEvent.TXInfo.Frequency),
			DR:        uplinkEvent.DR,
		},
	}

	if len(uplinkEvent.RXInfo) > 0 {
		e.Location = types.Location{} // no such information in ChirpStack v4 uplink event
		e.RX = &types.RX{
			RSSI:    float64(uplinkEvent.RXInfo[0].RSSI),
			LoRaSNR: uplinkEvent.RXInfo[0].SNR,
		}
	}

	return e, nil
}

type UplinkEvent struct {
	DeduplicationID string     `json:"deduplicationId"`
	Time            time.Time  `json:"time"`
	DeviceInfo      DeviceInfo `json:"deviceInfo"`
	DevAddr         string     `json:"devAddr"`
	DR              int        `json:"dr"`
	FPort           int        `json:"fPort"`
	FCnt            uint32     `json:"fCnt"`
	Data            string     `json:"data"`
	RXInfo          []RXInfo   `json:"rxInfo"`
	TXInfo          TXInfo     `json:"txInfo"`

	Object     json.RawMessage `json:"object"`
	ObjectJSON json.RawMessage `json:"objectJSON"`
}

type StatusEvent struct {
	DeduplicationID string     `json:"deduplicationId"`
	Time            time.Time  `json:"time"`
	DeviceInfo      DeviceInfo `json:"deviceInfo"`
	Margin          int        `json:"margin"`
	BatteryLevel    float64    `json:"batteryLevel"`
}

type LogEvent struct {
	Time        time.Time         `json:"time"`
	DeviceInfo  DeviceInfo        `json:"deviceInfo"`
	Level       string            `json:"level"`
	Code        string            `json:"code"`
	Description string            `json:"description"`
	Context     map[string]string `json:"context"`
}

type DeviceInfo struct {
	TenantID          string            `json:"tenantId"`
	TenantName        string            `json:"tenantName"`
	ApplicationID     string            `json:"applicationId"`
	ApplicationName   string            `json:"applicationName"`
	DeviceProfileID   string            `json:"deviceProfileId"`
	DeviceProfileName string            `json:"deviceProfileName"`
	DeviceName        string            `json:"deviceName"`
	DevEUI            string            `json:"devEui"`
	Tags              map[string]string `json:"tags"`
}

type RXInfo struct {
	GatewayID string            `json:"gatewayId"`
	UplinkID  uint32            `json:"uplinkId"`
	RSSI      int               `json:"rssi"`
	SNR       float64           `json:"snr"`
	Context   string            `json:"context"`
	Metadata  map[string]string `json:"metadata"`
}

type TXInfo struct {
	Frequency  uint64     `json:"frequency"`
	Modulation Modulation `json:"modulation"`
}

type Modulation struct {
	LoRa LoRaModulation `json:"lora"`
}

type LoRaModulation struct {
	Bandwidth       int    `json:"bandwidth"`
	SpreadingFactor int    `json:"spreadingFactor"`
	CodeRate        string `json:"codeRate"`
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
