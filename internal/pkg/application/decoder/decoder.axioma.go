package decoder

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"

	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
)

var ErrTimeTooFarOff = fmt.Errorf("sensor time is too far off in the future")

func Qalcosonic_Auto(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	var err error

	buf := bytes.NewReader(ue.Data)
	if buf.Len() < 42 {
		return fmt.Errorf("w1b decoder not implemented or payload to short")
	}
	var m measurementDecoder
	if buf.Len() == 51 || buf.Len() == 52 {
		m = w1e
	} else if buf.Len() <= 47 {
		m = w1h
	}

	err = qalcosonicW1(ctx, ue, m, fn)
	if err != nil && errors.Is(err, ErrTimeTooFarOff) {
		err = qalcosonicW1(ctx, ue, w1t, fn)
	}

	return err
}

func Qalcosonic_w1h(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	return qalcosonicW1(ctx, ue, w1h, fn)
}

func Qalcosonic_w1t(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	return qalcosonicW1(ctx, ue, w1t, fn)
}

func Qalcosonic_w1e(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	return qalcosonicW1(ctx, ue, w1e, fn)
}

func qalcosonicW1(ctx context.Context, ue mqtt.UplinkEvent, m measurementDecoder, fn func(context.Context, Payload) error) error {
	if ue.FPort != 100 {
		return fmt.Errorf("fPort %d not implemented", ue.FPort)
	}

	p := Payload{
		DevEUI:    ue.DevEui,
		Timestamp: ue.Timestamp.Format(time.RFC3339Nano),
	}

	buf := bytes.NewReader(ue.Data)
	if m, err := m(buf); err == nil {
		p.Measurements = append(p.Measurements, m...)
	} else {
		return fmt.Errorf("unable to decode measurements, %w", err)
	}

	code := p.ValueOf("StatusCode").(int)
	messages := p.ValueOf("StatusMessages").([]string)

	p.SetStatus(code, messages)

	err := fn(ctx, p)
	if err != nil {
		return err
	}

	return nil
}

type measurementDecoder = func(buf *bytes.Reader) ([]any, error)

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
		dateTime := time.Unix(int64(logDateTime), 0).UTC()
		now := time.Now().UTC()

		if dateTime.After(now.Add(-24*time.Hour)) && dateTime.Before(now.Add(24*time.Hour)) {
			dateTime = now
		} else if dateTime.After(now.Add(24 * time.Hour)) {
			return nil, ErrTimeTooFarOff
		}

		m := struct {
			LogDateTime string `json:"logDateTime"`
		}{
			LogDateTime: dateTime.Format(time.RFC3339Nano),
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

	if d, ok := deltaVolumes(buf, lastLogValue, logDateTime); ok {
		measurements = append(measurements, d...)
	}

	return measurements, nil
}

func w1t(buf *bytes.Reader) ([]interface{}, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var lastLogValueDate uint32
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

	err = binary.Read(buf, binary.LittleEndian, &temperature)
	if err == nil {
		m := struct {
			Temperature float64 `json:"temperature"`
		}{
			Temperature: float64(temperature) * 0.01,
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &lastLogValueDate)
	if err == nil {
		m := struct {
			LastLogValueDate string `json:"lastLogValueDate"`
		}{
			LastLogValueDate: time.Unix(int64(lastLogValueDate), 0).UTC().Format(time.RFC3339Nano),
		}
		measurements = append(measurements, m)
	} else {
		return nil, err
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

	if d, ok := deltaVolumes(buf, lastLogValue, lastLogValueDate); ok {
		measurements = append(measurements, d...)
	}

	return measurements, nil
}

func deltaVolumes(buf *bytes.Reader, lastLogValue, lastLogValueDate uint32) ([]interface{}, bool) {
	var deltaVolume uint16
	var measurements []interface{}

	deltas := struct {
		DeltaVolumes []struct {
			Volume       float64 `json:"volume"`
			Cumulated    float64 `json:"cumulated"`
			LogValueDate string  `json:"logValueDate"`
		} `json:"deltaVolumes"`
	}{}

	t := time.Unix(int64(lastLogValueDate), 0).UTC()
	v := lastLogValue

	for {
		err := binary.Read(buf, binary.LittleEndian, &deltaVolume)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		} else if err != nil {
			return nil, false
		}

		vol := struct {
			Volume       float64 `json:"volume"`
			Cumulated    float64 `json:"cumulated"`
			LogValueDate string  `json:"logValueDate"`
		}{
			Volume:       float64(deltaVolume) * 0.001,
			Cumulated:    float64(v+uint32(deltaVolume)) * 0.001,
			LogValueDate: t.Add(time.Hour).Format(time.RFC3339Nano),
		}

		t = t.Add(time.Hour)
		v = v + uint32(deltaVolume)

		deltas.DeltaVolumes = append(deltas.DeltaVolumes, vol)
	}

	measurements = append(measurements, deltas)

	return measurements, true
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
