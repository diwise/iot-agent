package decoder

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"

	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

var ErrTimeTooFarOff = fmt.Errorf("sensor time is too far off in the future")

func QalcosonicAuto(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	var err error

	buf := bytes.NewReader(ue.Data)
	if buf.Len() < 42 {
		return fmt.Errorf("w1b decoder not implemented or payload to short (%d)", buf.Len())
	}

	var m measurementDecoder
	if buf.Len() == 51 || buf.Len() == 52 {
		m = w1h
	} else if buf.Len() <= 47 {
		m = w1e
	}

	err = qalcosonicW1(ctx, ue, m, fn)
	if err != nil && errors.Is(err, ErrTimeTooFarOff) {
		err = qalcosonicW1(ctx, ue, w1t, fn)
	}

	return err
}

func QalcosonicW1e(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	return qalcosonicW1(ctx, ue, w1e, fn)
}

func QalcosonicW1t(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	return qalcosonicW1(ctx, ue, w1t, fn)
}

func QalcosonicW1h(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	return qalcosonicW1(ctx, ue, w1h, fn)
}

func qalcosonicW1(ctx context.Context, ue application.SensorEvent, measurementDecoder measurementDecoder, fn func(context.Context, payload.Payload) error) error {
	if ue.FPort != 100 {
		return fmt.Errorf("fPort %d not implemented", ue.FPort)
	}

	var decorators []payload.PayloadDecoratorFunc

	buf := bytes.NewReader(ue.Data)
	if d, err := measurementDecoder(buf); err == nil {
		decorators = append(decorators, d...)
	} else {
		return fmt.Errorf("unable to decode measurements, %w", err)
	}

	pp, _ := payload.New(ue.DevEui, ue.Timestamp, decorators...)

	return fn(ctx, pp)
}

type measurementDecoder = func(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error)

func w1e(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var logDateTime uint32
	var lastLogValue uint32

	var decorators []payload.PayloadDecoratorFunc

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime := time.Unix(int64(epoch), 0).UTC()

		if tooFarOff(sensorTime) {
			return nil, ErrTimeTooFarOff
		}

		decorators = append(decorators, payload.CurrentTime(sensorTime))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		decorators = append(decorators, payload.Status(statusCode, getStatusMessage(statusCode)))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err == nil {
		decorators = append(decorators, payload.CurrentVolume(float64(currentVolume)*0.001))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		dateTime := time.Unix(int64(logDateTime), 0).UTC()

		if tooFarOff(dateTime) {
			return nil, ErrTimeTooFarOff
		}

		decorators = append(decorators, payload.LogDateTime(dateTime))
	}

	err = binary.Read(buf, binary.LittleEndian, &lastLogValue)
	if err == nil {
		decorators = append(decorators, payload.LastLogValue(float64(lastLogValue)*0.001))
	} else {
		return nil, err
	}

	if d, ok := deltaVolumes(buf, lastLogValue, logDateTime); ok {
		decorators = append(decorators, d...)
	}

	return decorators, nil
}

func w1t(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var lastLogValueDate uint32
	var lastLogValue uint32

	var decorators []payload.PayloadDecoratorFunc

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime := time.Unix(int64(epoch), 0).UTC()

		if tooFarOff(sensorTime) {
			return nil, ErrTimeTooFarOff
		}

		decorators = append(decorators, payload.CurrentTime(sensorTime))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		decorators = append(decorators, payload.Status(statusCode, getStatusMessage(statusCode)))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err == nil {
		decorators = append(decorators, payload.CurrentVolume(float64(currentVolume)*0.001))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &temperature)
	if err == nil {
		decorators = append(decorators, payload.Temperature(float64(temperature)*0.01))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &lastLogValueDate)
	if err == nil {
		decorators = append(decorators, payload.LogDateTime(time.Unix(int64(lastLogValueDate), 0).UTC()))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &lastLogValue)
	if err == nil {
		decorators = append(decorators, payload.LastLogValue(float64(lastLogValue)*0.001))
	} else {
		return nil, err
	}

	if d, ok := deltaVolumes(buf, lastLogValue, lastLogValueDate); ok {
		decorators = append(decorators, d...)
	}

	return decorators, nil
}

func deltaVolumes(buf *bytes.Reader, lastLogValue, lastLogValueDate uint32) ([]payload.PayloadDecoratorFunc, bool) {
	var deltaVolume uint16
	var decorators []payload.PayloadDecoratorFunc

	t := time.Unix(int64(lastLogValueDate), 0).UTC()
	v := lastLogValue

	for {
		err := binary.Read(buf, binary.LittleEndian, &deltaVolume)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		} else if err != nil {
			return nil, false
		}

		decorators = append(decorators, payload.DeltaVolume(float64(deltaVolume)*0.001, float64(v+uint32(deltaVolume))*0.001, t.Add(time.Hour)))

		t = t.Add(time.Hour)
		v = v + uint32(deltaVolume)
	}

	return decorators, true
}

func w1h(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var frameVersion uint8
	var epoch uint32
	var statusCode uint8
	var currentVolume uint32

	var decorators []payload.PayloadDecoratorFunc

	err = binary.Read(buf, binary.LittleEndian, &frameVersion)
	if err == nil {
		decorators = append(decorators, payload.FrameVersion(frameVersion))
	} else {
		return nil, err
	}

	var sensorTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return nil, ErrTimeTooFarOff
		}
		decorators = append(decorators, payload.CurrentTime(sensorTime))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		decorators = append(decorators, payload.Status(statusCode, getStatusMessage(statusCode)))
	} else {
		return nil, err
	}

	var currVol float64 = 0.0
	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err == nil {
		currVol = float64(currentVolume) * 0.001
		decorators = append(decorators, payload.CurrentVolume(currVol))
	} else {
		return nil, err
	}

	if d, ok := deltaVolumesH(buf, currVol, sensorTime); ok {
		decorators = append(decorators, d...)
	}

	return decorators, nil
}

func deltaVolumesH(buf *bytes.Reader, currentVolume float64, sensorTime time.Time) ([]payload.PayloadDecoratorFunc, bool) {
	var decorators []payload.PayloadDecoratorFunc
	data, _ := io.ReadAll(buf)
	data = append(data, 0)

	decode := func(input []byte) []uint64 {
		result := make([]uint64, 0, 24)
		for len(input) >= 7 {
			quad := input[0:7]
			input = input[7:]
			result = append(
				result,
				((uint64(quad[1])<<8)%16384)|uint64(quad[0]),
				((uint64(quad[3])<<10)%16384)|uint64(quad[2])<<2|uint64(quad[1])>>6,
				((uint64(quad[5])<<12)%16384)|uint64(quad[4])<<4|uint64(quad[3])>>4,
				(uint64(quad[6])<<6)|uint64(quad[5])>>2,
			)
		}
		return result
	}

	deltas := decode(data)
	totalVol := currentVolume
	deltaTime := sensorTime

	for i := 0; i < 23; i++ {
		totalVol += float64(deltas[i])*0.001
		deltaTime = deltaTime.Add(1 * time.Hour)
		dec := payload.DeltaVolume(float64(deltas[i])*0.001, totalVol, deltaTime)
		decorators = append(decorators, dec)
	}

	return decorators, len(decorators) > 0
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

func tooFarOff(t time.Time) bool {
	return t.After(time.Now().UTC().Add(72 * time.Hour))
}
