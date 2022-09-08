package decoder

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"strings"

	"encoding/json"
	"fmt"
	"time"
)

func AxiomaWatermeteringDecoder(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {
	d := struct {
		DevEUI            string      `json:"devEui"`
		SensorType        *string     `json:"sensorType,omitempty"`
		DeviceName        *string     `json:"deviceName,omitempty"`
		DeviceProfileName *string     `json:"deviceProfileName,omitempty"`
		Timestamp         string      `json:"timestamp"`
		BatteryLevel      string      `json:"batteryLevel"`
		FPort             interface{} `json:"fPort"`
		Payload           *string     `json:"payload,omitempty"`
		Data              *string     `json:"data,omitempty"`
	}{}

	err := json.Unmarshal(msg, &d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %s", err.Error())
	}

	q := strings.Trim(fmt.Sprintf("%v", d.FPort), "\"")
	if q != "100" {
		return fmt.Errorf("only incomming messages with fPort 100 are supported")
	}

	var sensorType *string
	if d.SensorType != nil {
		sensorType = d.SensorType
	} else if d.DeviceProfileName != nil {
		sensorType = d.DeviceProfileName
	} else {
		return fmt.Errorf("unable to resolve type of sensor")
	}

	var deviceName string
	if d.DeviceName == nil {
		deviceName = *sensorType
	} else {
		deviceName = *d.DeviceName
	}

	payload := Payload{
		DevEUI:     d.DevEUI,
		DeviceName: deviceName,
		FPort:      q,
		SensorType: *sensorType,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	var buf *bytes.Reader = nil

	if d.Payload != nil {
		b, err := readPayload(*d.Payload)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)

	} else if d.Data != nil {
		b, err := readPayload(*d.Data)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}

	if buf == nil {
		return fmt.Errorf("unable to read payload")
	}

	if buf.Len() < 42 {
		return fmt.Errorf("short decoder not implemented")
	}

	if strings.EqualFold(*sensorType, "qalcosonic_w1h") {
		measurements, err := w1h(buf)
		if err != nil {
			return err
		}
		payload.Measurements = append(payload.Measurements, measurements...)

	} else if strings.EqualFold(*sensorType, "qalcosonic_w1h_temp") {
		measurements, err := w1hTemp(buf)
		if err != nil {
			return err
		}
		payload.Measurements = append(payload.Measurements, measurements...)

	} else {
		measurements, err := w24h(buf)
		if err != nil {
			return err
		}
		payload.Measurements = append(payload.Measurements, measurements...)
	}

	payload.StatusCode = payload.ValueOf("StatusCode").(int)

	err = fn(ctx, payload)
	if err != nil {
		return err
	}

	return nil
}

func w1h(buf *bytes.Reader) ([]interface{}, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var logDateTime uint32
	var lastLogValue uint32

	var measurements []interface{}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		currentTime := time.Unix(int64(epoch), 0)
		if currentTime.After(time.Now().UTC()) {
			return nil, fmt.Errorf("invalid time")
		}

		m := struct {
			CurrentTime string `json:"currentTime"`
		}{
			CurrentTime: currentTime.Format(time.RFC3339),
		}

		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		m := struct {
			StatusCode     int      `json:"statusCode"`
			StatusMessages []string `json:"statusMessages"`
		}{
			StatusCode:     int(statusCode),
			StatusMessages: getStatusMessage(statusCode),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err == nil {
		m := struct {
			CurrentVolume float64 `json:"currentVolume"`
		}{
			CurrentVolume: float64(currentVolume) * 0.001,
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		m := struct {
			LogDateTime string `json:"logDateTime"`
		}{
			LogDateTime: time.Unix(int64(logDateTime), 0).Format(time.RFC3339),
		}
		measurements = append(measurements, m)
	}

	err = binary.Read(buf, binary.LittleEndian, &lastLogValue)
	if err == nil {
		m := struct {
			LastLogValue float64 `json:"lastLogValue"`
		}{
			LastLogValue: float64(lastLogValue) * 0.001,
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	return measurements, nil
}

func w1hTemp(buf *bytes.Reader) ([]interface{}, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var logDateTime uint32

	var measurements []interface{}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		currentTime := time.Unix(int64(epoch), 0)
		if currentTime.After(time.Now().UTC()) {
			return nil, fmt.Errorf("invalid time")
		}

		m := struct {
			CurrentTime string `json:"currentTime"`
		}{
			CurrentTime: currentTime.Format(time.RFC3339),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		m := struct {
			StatusCode     int      `json:"statusCode"`
			StatusMessages []string `json:"statusMessages"`
		}{
			StatusCode:     int(statusCode),
			StatusMessages: getStatusMessage(statusCode),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err == nil {
		m := struct {
			CurrentVolume float64 `json:"currentVolume"`
		}{
			CurrentVolume: float64(currentVolume) * 0.001,
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &temperature)
	if err == nil {
		m := struct {
			Temperature float64 `json:"temperature"`
		}{
			Temperature: float64(temperature) * 0.001,
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		m := struct {
			LogDateTime string `json:"logDateTime"`
		}{
			LogDateTime: time.Unix(int64(logDateTime), 0).Format(time.RFC3339),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	return measurements, nil
}

func w24h(buf *bytes.Reader) ([]interface{}, error) {
	var err error

	var frameVersion uint8
	var epoch uint32
	var statusCode uint8
	var currentVolume uint32

	var measurements []interface{}

	err = binary.Read(buf, binary.LittleEndian, &frameVersion)
	if err == nil {
		m := struct {
			FrameVersion int `json:"frameVersion"`
		}{
			FrameVersion: int(frameVersion),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		currentTime := time.Unix(int64(epoch), 0)
		if currentTime.After(time.Now().UTC()) {
			return nil, fmt.Errorf("invalid time")
		}

		m := struct {
			CurrentTime string `json:"currentTime"`
		}{
			CurrentTime: currentTime.Format(time.RFC3339),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		m := struct {
			StatusCode     int      `json:"statusCode"`
			StatusMessages []string `json:"statusMessages"`
		}{
			StatusCode:     int(statusCode),
			StatusMessages: getStatusMessage(statusCode),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err == nil {
		m := struct {
			CurrentVolume float64 `json:"currentVolume"`
		}{
			CurrentVolume: float64(currentVolume) * 0.001,
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	return measurements, nil
}

func readPayload(payload string) ([]byte, error) {
	b, err := hex.DecodeString(payload)
	if err == nil {
		return b, nil
	}

	b, err = base64.RawStdEncoding.DecodeString(payload)
	if err == nil {
		return b, nil
	}

	return nil, err
}

func getStatusMessage(code uint8) []string {
	var statusMessages []string

	if code == 0x00 {
		statusMessages = append(statusMessages, "No error")
	} else {
		if code&0x04 == 0x04 {
			statusMessages = append(statusMessages, "Power low")
		}
		if code&0x08 == 0x08 {
			statusMessages = append(statusMessages, "Permanent error")
		}
		if code&0x10 == 0x10 {
			statusMessages = append(statusMessages, "Temporary error")
		}
		if code&0x10 == 0x10 && code&0x20 != 0x20 && code&0xA0 != 0xA0 && code&0x60 != 0x60 && code&0x80 != 0x80 {
			statusMessages = append(statusMessages, "Empty spool")
		}
		if code&0x60 == 0x60 {
			statusMessages = append(statusMessages, "Negative flow")
		}
		if code&0xA0 == 0xA0 {
			statusMessages = append(statusMessages, "Burst")
		}
		if code&0x20 == 0x20 && code&0x40 != 0x40 && code&0x80 != 0x80 {
			statusMessages = append(statusMessages, "Leakage")
		}
		if code&0x80 == 0x80 && code&0x20 != 0x20 {
			statusMessages = append(statusMessages, "Freeze")
		}
	}

	if len(statusMessages) == 0 {
		statusMessages = append(statusMessages, "Unknown")
	}

	return statusMessages
}
