package servanet

import (
	"encoding/base64"
	"encoding/json"
	"time"

	. "github.com/diwise/iot-agent/internal/pkg/application/types"
)

func HandleUplinkEvent(b []byte) (SensorEvent, error) {
	mapToMapArr := func(m map[string]string) map[string][]string {
		ma := make(map[string][]string, 0)

		if m == nil {
			return ma
		}

		for k, v := range m {
			ma[k] = []string{v}
		}

		return ma
	}

	var uplinkEvent UplinkEvent
	err := json.Unmarshal(b, &uplinkEvent)
	if err != nil {
		return SensorEvent{}, err
	}

	var data []byte
	data, err = base64.StdEncoding.DecodeString(uplinkEvent.Data)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(uplinkEvent.Data)
		if err != nil {
			return SensorEvent{}, err
		}
	}

	if uplinkEvent.Data == "" && uplinkEvent.Object == nil {
		return SensorEvent{}, ErrPayloadContainsNoData
	}

	var objectJSON json.RawMessage
	if uplinkEvent.Object != nil {
		objectJSON = uplinkEvent.Object
	} else {
		objectJSON = uplinkEvent.ObjectJSON
	}

	ue := SensorEvent{
		SensorType: uplinkEvent.DeviceProfileName,
		DeviceName: uplinkEvent.DeviceName,
		DevEui:     uplinkEvent.DevEUI,
		FPort:      uint8(uplinkEvent.FPort),
		Data:       data,
		Object:     objectJSON,
		Tags:       mapToMapArr(uplinkEvent.Tags),
	}

	if len(uplinkEvent.RxInfo) > 0 {
		ue.RXInfo = RXInfo{
			GatewayId: uplinkEvent.RxInfo[0].GatewayID,
			UplinkId:  uplinkEvent.RxInfo[0].UplinkID,
			Rssi:      float64(uplinkEvent.RxInfo[0].RSSI),
			Snr:       uplinkEvent.RxInfo[0].LoRaSNR,
		}
	}

	ue.Timestamp = time.Now().UTC()

	return ue, nil
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
	ApplicationID           string  `json:"applicationID"`
	ApplicationName         string  `json:"applicationName"`
	DeviceName              string  `json:"deviceName"`
	DevEUI                  string  `json:"devEUI"`
	Margin                  int     `json:"margin"`
	ExternalPowerSource     bool    `json:"externalPowerSource"`
	BatteryLevel            float64 `json:"batteryLevel"`
	BatteryLevelUnavailable bool    `json:"batteryLevelUnavailable"`
}
