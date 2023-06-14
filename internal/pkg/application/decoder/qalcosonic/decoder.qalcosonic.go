package qalcosonic

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

func W1Decoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	var err error

	buf := bytes.NewReader(ue.Data)

	var m measurementDecoder

	if buf.Len() == 5 {
		m = alarmPacketDecoder
	} else if buf.Len() < 42 {
		return errors.New("decoder not implemented or payload to short")
	} else if buf.Len() == 51 || buf.Len() == 52 {
		m = w1h
	} else if buf.Len() <= 47 {
		m = w1e
	}

	err = decodeQalcosonicPayload(ctx, ue, m, fn)
	if err != nil && errors.Is(err, ErrTimeTooFarOff) {
		err = decodeQalcosonicPayload(ctx, ue, w1t, fn)
	}

	return err
}

func decodeQalcosonicPayload(ctx context.Context, ue application.SensorEvent, measurementDecoder measurementDecoder, fn func(context.Context, payload.Payload) error) error {
	if !(ue.FPort == 100 || ue.FPort == 103) {
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

func alarmPacketDecoder(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var epoch uint32
	var statusCode uint8

	var decorators []payload.PayloadDecoratorFunc

	var sensorTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err != nil {
		return nil, err
	}

	sensorTime = time.Unix(int64(epoch), 0).UTC()
	decorators = append(decorators, payload.Timestamp(sensorTime))

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err != nil {
		return nil, err
	}

	var statusMessages []string
	if statusMessages = getStatusMessageForAlarmPacket(statusCode); statusMessages == nil {
		statusMessages = getStatusMessage(statusCode)
	}

	return append(decorators, payload.Status(statusCode, statusMessages)), nil
}

// Lora Payload (24 hours) “Enhanced”
func w1e(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var logDateTime uint32
	var volumeAtLogDateTime uint32

	var decorators []payload.PayloadDecoratorFunc

	var sensorTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return nil, ErrTimeTooFarOff
		}
		decorators = append(decorators, payload.Timestamp(sensorTime))
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
	if err != nil {
		return nil, err
	}

	var ldt time.Time
	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		dt := time.Unix(int64(logDateTime), 0).UTC()
		if tooFarOff(dt) {
			return nil, ErrTimeTooFarOff
		}

		// Log values are always equal to beginning of an hour or a day
		hh, _, _ := dt.Clock()
		y, m, d := dt.Date()
		ldt = time.Date(y, m, d, hh, 0, 0, 0, time.UTC)
	}

	err = binary.Read(buf, binary.LittleEndian, &volumeAtLogDateTime)
	if err != nil {
		return nil, err
	}

	decorators = append(decorators, payload.Volume(0, float64(volumeAtLogDateTime), ldt))

	if d, ok := deltaVolumes(buf, volumeAtLogDateTime, ldt); ok {
		decorators = append(decorators, d...)
	}

	decorators = append(decorators, payload.Volume(0, float64(currentVolume), sensorTime))

	decorators = append(decorators, payload.Type("w1e"))

	return decorators, nil
}

func w1t(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var logDateTime uint32
	var volumeAtLogDateTime uint32

	var decorators []payload.PayloadDecoratorFunc

	var sensorTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return nil, ErrTimeTooFarOff
		}
		decorators = append(decorators, payload.Timestamp(sensorTime))
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
	if err != nil {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &temperature)
	if err != nil {
		return nil, err
	}

	var ldt time.Time
	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		// Log values are always equal to beginning of an hour or a day
		dt := time.Unix(int64(logDateTime), 0).UTC()
		hh, _, _ := dt.Clock()
		y, m, d := dt.Date()
		ldt = time.Date(y, m, d, hh, 0, 0, 0, time.UTC)
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &volumeAtLogDateTime)
	if err != nil {
		return nil, err
	}

	decorators = append(decorators, payload.Volume(0, float64(volumeAtLogDateTime), ldt))

	if d, ok := deltaVolumes(buf, volumeAtLogDateTime, ldt); ok {
		decorators = append(decorators, d...)
	}

	decorators = append(decorators, payload.Volume(0, float64(currentVolume), sensorTime))

	decorators = append(decorators, payload.Temperature(float64(temperature)))
	decorators = append(decorators, payload.Type("w1t"))

	return decorators, nil
}

func deltaVolumes(buf *bytes.Reader, lastLogValue uint32, logDateTime time.Time) ([]payload.PayloadDecoratorFunc, bool) {
	var deltaVolume uint16
	var decorators []payload.PayloadDecoratorFunc

	t := logDateTime
	v := lastLogValue

	for {
		err := binary.Read(buf, binary.LittleEndian, &deltaVolume)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		} else if err != nil {
			return nil, false
		}

		decorators = append(decorators, payload.Volume(float64(deltaVolume), float64(v+uint32(deltaVolume)), t.Add(time.Hour)))

		t = t.Add(time.Hour)
		v = v + uint32(deltaVolume)
	}

	return decorators, true
}

// Lora Payload (Long) “Extended”
func w1h(buf *bytes.Reader) ([]payload.PayloadDecoratorFunc, error) {
	var err error

	var frameVersion uint8
	var epoch uint32
	var statusCode uint8
	var logVolumeAtOne uint32

	var decorators []payload.PayloadDecoratorFunc

	err = binary.Read(buf, binary.LittleEndian, &frameVersion)
	if err == nil {
		decorators = append(decorators, payload.FrameVersion(frameVersion))
	} else {
		return nil, err
	}

	var sensorTime time.Time
	var logDateTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return nil, ErrTimeTooFarOff
		}
		// First full value is logged at 01:00 time. Other 23 values are differences (increments).
		// All values are logged and always equal to beginning of an hour or a day
		y, m, d := sensorTime.Date()
		logDateTime = time.Date(y, m, d-1, 1, 0, 0, 0, time.UTC)
		decorators = append(decorators, payload.Timestamp(sensorTime))
	} else {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		decorators = append(decorators, payload.Status(statusCode, getStatusMessage(statusCode)))
	} else {
		return nil, err
	}

	var vol float64 = 0.0
	err = binary.Read(buf, binary.LittleEndian, &logVolumeAtOne)
	if err == nil {
		vol = float64(logVolumeAtOne)

	} else {
		return nil, err
	}

	decorators = append(decorators, payload.Volume(0, float64(vol), logDateTime))

	if d, ok := deltaVolumesH(buf, vol, logDateTime); ok {
		decorators = append(decorators, d...)
	}

	decorators = append(decorators, payload.Type("w1h"))

	return decorators, nil
}

func deltaVolumesH(buf *bytes.Reader, currentVolume float64, logTime time.Time) ([]payload.PayloadDecoratorFunc, bool) {
	var decorators []payload.PayloadDecoratorFunc
	data, _ := io.ReadAll(buf)
	data = append(data, 0) // append 0 for last quad

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
	deltaTime := logTime

	for i := 0; i < 23; i++ {
		totalVol += float64(deltas[i])
		deltaTime = deltaTime.Add(1 * time.Hour)
		decorators = append(decorators, payload.Volume(float64(deltas[i]), float64(totalVol), deltaTime))
	}

	return decorators, len(decorators) > 0
}

func getStatusMessage(code uint8) []string {
	const (
		NoError        = 0x00
		PowerLow       = 0x04
		PermanentError = 0x08
		TemporaryError = 0x10
		EmptySpool     = 0x10 // same as Temporary Error
		Leak           = 0x20
		Burst          = 0xA0
		Backflow       = 0x60 // negative flow
		Freeze         = 0x80
	)

	msg := make([]string, 0)

	if code == NoError {
		msg = append(msg, "No error")
	}

	if code&PowerLow == PowerLow {
		msg = append(msg, "Power low")
	}

	if code&PermanentError == PermanentError {
		msg = append(msg, "Permanent error")
	}

	if code&TemporaryError == TemporaryError {
		msg = append(msg, "Temporary error")
	}

	// If status only shows temporary error, it is empty spool.
	if code&EmptySpool == EmptySpool && code&Freeze != Freeze && code&Leak != Leak && code&Burst != Burst && code&Backflow != Backflow {
		msg = append(msg, "Empty spool")
	}

	// priority: freeze; leakage; burst; negative flow
	if code&Freeze == Freeze && code&Leak != Leak && code&Burst != Burst && code&Backflow != Backflow {
		msg = append(msg, "Freeze")
	}

	if code&Leak == Leak && code&Freeze != Freeze && code&Burst != Burst && code&Backflow != Backflow {
		msg = append(msg, "Leak")
	}

	if code&Burst == Burst {
		msg = append(msg, "Burst")
	}

	if code&Backflow == Backflow {
		msg = append(msg, "Backflow")
	}

	if len(msg) == 0 {
		msg = append(msg, "Unknown")
	}

	return msg
}

func getStatusMessageForAlarmPacket(code uint8) []string {
	const (
		NoError        = 0x00
		Leakage        = 0x01
		Burst          = 0x02
		LowTemperature = 0x04
		Tamper         = 0x08
		//NoConsumption  = 0x10
		NegativeFlow = 0x20
	)

	if code == NoError {
		return []string{"No error"}
	}

	if code == Leakage {
		return []string{"Leak"}
	}

	if code == Burst {
		return []string{"Burst"}
	}

	if code == LowTemperature {
		return []string{"Low temperature"}
	}

	if code == Tamper {
		return []string{"Tamper"}
	}

	// off by default
	//if code == NoConsumption {
	//	return []string{"No consumption"}
	//}

	if code == NegativeFlow {
		return []string{"Backflow"}
	}

	return nil
}

func tooFarOff(t time.Time) bool {
	return t.After(time.Now().UTC().Add(72 * time.Hour))
}
