package decoder

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"

	"encoding/json"
	"fmt"
	"time"
)

var ErrTimeTooFarOff = fmt.Errorf("sensor time is too far off in the future")

func AxiomaWatermeteringDecoder(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {
	payload, buf, err := initialize(msg)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	if buf.Len() < 42 {
		return fmt.Errorf("w1b decoder not implemented")
	}

	if buf.Len() == 51 || buf.Len() == 52 {
		measurements, err := w1e(buf)
		if err != nil {
			return fmt.Errorf("unable to decode w1e measurements")
		}
		payload.Measurements = append(payload.Measurements, measurements...)
	} else if buf.Len() <= 47 {
		measurements, err := w1h(buf)
		if err != nil {
			if errors.Is(err, ErrTimeTooFarOff) {
				measurements, err = w1t(buf)
				if err != nil {
					return fmt.Errorf("unable to decode w1t measurements")
				}
			} else {
				return fmt.Errorf("unable to decode w1h measurements")
			}
		}
		payload.Measurements = append(payload.Measurements, measurements...)
	} else {
		return fmt.Errorf("unable to resolve decoder")
	}

	code := payload.ValueOf("StatusCode").(int)
	messages := payload.ValueOf("StatusMessages").([]string)

	payload.SetStatus(code, messages)

	err = fn(ctx, *payload)
	if err != nil {
		return err
	}

	return nil
}

func Qalcosonic_w1h(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {
	return qalcosonic(ctx, msg, w1h, fn)
}

func Qalcosonic_w1t(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {
	return qalcosonic(ctx, msg, w1t, fn)
}

func Qalcosonic_w1e(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {
	return qalcosonic(ctx, msg, w1e, fn)
}

func w1h(buf *bytes.Reader) ([]any, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var logDateTime uint32
	var lastLogValue uint32

	var measurements []interface{}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime := time.Unix(int64(epoch), 0).UTC()
		now := time.Now().UTC()

		// Handle clock skew by setting sensor time to current time if it is from
		// within a 48 hour window around current time
		if sensorTime.After(now.Add(-24*time.Hour)) && sensorTime.Before(now.Add(24*time.Hour)) {
			sensorTime = now
		} else if sensorTime.After(now.Add(24 * time.Hour)) {
			return nil, ErrTimeTooFarOff
		}

		m := struct {
			CurrentTime string `json:"currentTime"`
		}{
			CurrentTime: sensorTime.Format(time.RFC3339Nano),
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

func w1t(buf *bytes.Reader) ([]interface{}, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var logDateTime uint32

	var measurements []interface{}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime := time.Unix(int64(epoch), 0).UTC()
		now := time.Now().UTC()

		// Handle clock skew by setting sensor time to current time if it is from
		// within a 48 hour window around current time
		if sensorTime.After(now.Add(-24*time.Hour)) && sensorTime.Before(now.Add(24*time.Hour)) {
			sensorTime = now
		} else if sensorTime.After(now.Add(24 * time.Hour)) {
			return nil, ErrTimeTooFarOff
		}

		m := struct {
			CurrentTime string `json:"currentTime"`
		}{
			CurrentTime: sensorTime.Format(time.RFC3339Nano),
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

func w1e(buf *bytes.Reader) ([]interface{}, error) {
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
		sensorTime := time.Unix(int64(epoch), 0).UTC()
		now := time.Now().UTC()

		// Handle clock skew by setting sensor time to current time if it is from
		// within a 48 hour window around current time
		if sensorTime.After(now.Add(-24*time.Hour)) && sensorTime.Before(now.Add(24*time.Hour)) {
			sensorTime = now
		} else if sensorTime.After(now.Add(24 * time.Hour)) {
			return nil, ErrTimeTooFarOff
		}

		m := struct {
			CurrentTime string `json:"currentTime"`
		}{
			CurrentTime: sensorTime.Format(time.RFC3339Nano),
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

func initialize(msg []byte) (*Payload, *bytes.Reader, error) {
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
		return nil, nil, fmt.Errorf("failed to unmarshal payload: %s", err.Error())
	}

	q := strings.Trim(fmt.Sprintf("%v", d.FPort), "\"")
	if q != "100" {
		return nil, nil, fmt.Errorf("only incomming messages with fPort 100 are supported")
	}

	var sensorType *string
	if d.SensorType != nil {
		sensorType = d.SensorType
	} else if d.DeviceProfileName != nil {
		sensorType = d.DeviceProfileName
	} else {
		return nil, nil, fmt.Errorf("unable to resolve type of sensor")
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
			return nil, nil, err
		}
		buf = bytes.NewReader(b)

	} else if d.Data != nil {
		b, err := readPayload(*d.Data)
		if err != nil {
			return nil, nil, err
		}
		buf = bytes.NewReader(b)
	}

	if buf == nil {
		return nil, nil, fmt.Errorf("unable to read payload")
	}

	return &payload, buf, nil
}

func qalcosonic(ctx context.Context, msg []byte, decoder func(buf *bytes.Reader) ([]any, error), fn func(context.Context, Payload) error) error {
	payload, buf, err := initialize(msg)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	if buf.Len() < 42 {
		return fmt.Errorf("short decoder not implemented")
	}

	measurements, err := decoder(buf)
	if err != nil {
		return fmt.Errorf("unable to decode measurements")
	}
	payload.Measurements = append(payload.Measurements, measurements...)

	code := payload.ValueOf("StatusCode").(int)
	messages := payload.ValueOf("StatusMessages").([]string)

	payload.SetStatus(code, messages)

	err = fn(ctx, *payload)
	if err != nil {
		return err
	}

	return nil
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
			statusMessages = append(statusMessages, "Backflow")
		}
		if code&0xA0 == 0xA0 {
			statusMessages = append(statusMessages, "Burst")
		}
		if code&0x20 == 0x20 && code&0x40 != 0x40 && code&0x80 != 0x80 {
			statusMessages = append(statusMessages, "Leak")
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
