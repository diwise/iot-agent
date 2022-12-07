package application

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type UplinkASFunc func([]byte) (SensorEvent, error)

func GetFacade(as string) UplinkASFunc {
	if strings.EqualFold("chirpstack", as) {
		return ChirpStack
	}
	if strings.EqualFold("netmore", as) {
		return Netmore
	}

	return ChirpStack
}

func ChirpStack(uplinkPayload []byte) (SensorEvent, error) {
	m := struct {
		ApplicationId     string          `json:"applicationId,omitempty"`
		ApplicationName   string          `json:"applicationName,omitempty"`
		DeviceProfileId   string          `json:"deviceProfileId"`
		DeviceProfileName string          `json:"deviceProfileName"`
		DeviceName        string          `json:"deviceName"`
		DevEui            string          `json:"devEui"`
		Data              string          `json:"data"`
		Object            json.RawMessage `json:"object,omitempty"`
		ObjectJSON        json.RawMessage `json:"objectJSON,omitempty"`
		FPort             uint8           `json:"fPort"`
		RXInfo            []struct {
			GatewayID string  `json:"gatewayID"`
			UplinkID  string  `json:"uplinkID"`
			Time      string  `json:"time"`
			Rssi      int32   `json:"rssi"`
			LoRaSNR   float32 `json:"loRaSNR"`
		} `json:"rxInfo,omitempty"`
		TXInfo struct {
			Frequency uint32 `json:"frequency"`
		}
		Tags map[string]string `json:"tags"`
		// only found in error messages
		Type  *string `json:"type,omitempty"`
		Error *string `json:"error,omitempty"`
	}{}

	err := json.Unmarshal(uplinkPayload, &m)
	if err != nil {
		return SensorEvent{}, err
	}

	var bytes []byte
	bytes, err = base64.StdEncoding.DecodeString(m.Data)
	if err != nil {
		bytes, err = base64.RawStdEncoding.DecodeString(m.Data)
		if err != nil {
			return SensorEvent{}, err
		}
	}

	if m.Data == "" && m.Object == nil {
		if m.Type == nil && m.Error == nil {
			t := "payload error"
			e := "uplink message contains no payload"
			m.Type = &t
			m.Error = &e
		}
	}

	var objectJSON json.RawMessage
	if m.Object != nil {
		objectJSON = m.Object
	} else {
		objectJSON = m.ObjectJSON
	}

	ue := SensorEvent{
		SensorType: m.DeviceProfileName,
		DeviceName: m.DeviceName,
		DevEui:     m.DevEui,
		FPort:      m.FPort,
		Data:       bytes,
		Object:     objectJSON,
		Tags:       mapToMapArr(m.Tags),
	}

	if m.RXInfo != nil && len(m.RXInfo) > 0 {
		ue.Timestamp = atot(m.RXInfo[0].Time)
		ue.RXInfo = RXInfo{
			GatewayId: m.RXInfo[0].GatewayID,
			UplinkId:  m.RXInfo[0].UplinkID,
			Rssi:      m.RXInfo[0].Rssi,
			Snr:       m.RXInfo[0].LoRaSNR,
		}
	} else {
		ue.Timestamp = time.Now().UTC()
	}

	if m.Type != nil || m.Error != nil {
		ue.Error = Error{
			Type:    *m.Type,
			Message: *m.Error,
		}
	}

	return ue, nil
}

func Netmore(uplinkPayload []byte) (SensorEvent, error) {
	m := []struct {
		DevEui            string              `json:"devEui"`
		SensorType        string              `json:"sensorType"`
		Timestamp         string              `json:"timestamp"`
		Data              string              `json:"payload"`
		SpreadingFactor   string              `json:"spreadingFactor"`
		Rssi              string              `json:"rssi"`
		Snr               string              `json:"snr"`
		GatewayIdentifier string              `json:"gatewayIdentifier"`
		FPort             string              `json:"fPort"`
		Frequency         uint32              `json:"freq"`
		Tags              map[string][]string `json:"tags"`
	}{}

	err := json.Unmarshal(uplinkPayload, &m)
	if err != nil {
		return SensorEvent{}, err
	}

	var bytes []byte
	bytes, err = hex.DecodeString(m[0].Data)
	if err != nil {
		return SensorEvent{}, err
	}

	ue := SensorEvent{
		DevEui:     m[0].DevEui,
		DeviceName: m[0].SensorType, // no DeviceName is received from Netmore
		SensorType: m[0].SensorType,
		FPort:      atoi[uint8](m[0].FPort),
		Data:       bytes,
		RXInfo: RXInfo{
			GatewayId: m[0].GatewayIdentifier,
			Rssi:      atoi[int32](m[0].Rssi),
			Snr:       atof(m[0].Snr),
		},
		TXInfo: TXInfo{
			Frequency: m[0].Frequency,
		},
		Tags:      m[0].Tags,
		Timestamp: atot(m[0].Timestamp),
	}

	return ue, nil
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

func atot(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}

func atoi[T int | int32 | uint8 | uint32](s string) T {
	if i, err := strconv.Atoi(s); err == nil {
		return T(i)
	}
	return 0
}

func atof[T float32](s string) T {
	if i, err := strconv.ParseFloat(s, 64); err == nil {
		return T(i)
	}
	return 0.0
}
