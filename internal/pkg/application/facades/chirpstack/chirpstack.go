package chirpstack

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
)

func HandleEvent(messageType string, b []byte) (types.Event, error) {
	return types.Event{}, errors.New("not implemented")
}

// UplinkEvent motsvarar ett fullständigt uplink-meddelande från ChirpStack v3.
type UplinkEvent struct {
	// Grundinformation
	ApplicationID   string `json:"applicationID"` // I prototexten uint64, men ofta skickat som string i JSON
	ApplicationName string `json:"applicationName"`
	DeviceName      string `json:"deviceName"`
	DevEUI          string `json:"devEUI"` // Base64‑kodat värde

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
	DevAddr           string    `json:"dev_addr"`     // Base64‑kodat
	PublishedAt       time.Time `json:"published_at"` // RFC3339-format
	DeviceProfileID   string    `json:"deviceProfileID"`
	DeviceProfileName string    `json:"deviceProfileName"`
}

// RxInfo representerar ett mottaget RX-objekt (gateway-specifik metadata).
type RxInfo struct {
	GatewayID         string    `json:"gatewayID"` // Base64‑kodat
	Time              time.Time `json:"time"`      // RFC3339‑formatat datum/tid
	TimeSinceGPSEpoch *int64    `json:"timeSinceGPSEpoch"`
	RSSI              int       `json:"rssi"`
	LoRaSNR           float64   `json:"loRaSNR"`
	Channel           int       `json:"channel"`
	RFChain           int       `json:"rfChain"`
	Board             int       `json:"board"`
	Antenna           int       `json:"antenna"`
	Location          Location  `json:"location"`
	FineTimestampType string    `json:"fineTimestampType"`
	Context           string    `json:"context"`  // Base64‑kodat
	UplinkID          string    `json:"uplinkID"` // Base64‑kodat
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
