package qalcosonic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"strings"

	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

var ErrTimeTooFarOff = fmt.Errorf("sensor time is too far off in the future")

const NoError = 0x00

type QalcosonicVolumePayload struct {
	CurrentVolume float64
	Deltas        []QalcosonicDeltaVolume
	FrameVersion  uint8
	Messages      []string
	StatusCode    uint8
	Temperature   *uint16
	Timestamp     time.Time
	Type          string
}

type QalcosonicAlarmPayload struct {
	Timestamp  time.Time
	StatusCode uint8
	Messages   []string
}

type QalcosonicPayload struct {
	volume *QalcosonicVolumePayload
	alarms *QalcosonicAlarmPayload
}

func (a QalcosonicPayload) BatteryLevel() *int {
	return nil
}
func (a QalcosonicPayload) Error() (string, []string) {
	if len(a.volume.Messages) > 0 {
		m := []string{}
		for _, v := range a.volume.Messages {
			if v != "No error" {
				m = append(m, v)
			}
		}

		return "", m
	}
	return "", []string{}
}

type QalcosonicDeltaVolume struct {
	CumulatedVolume float64
	DeltaVolume     float64
	Timestamp       time.Time
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	var err error

	p, ap, err := decodePayload(ctx, e)
	if err != nil {
		return nil, err
	}
	/*
		if p != nil && p.StatusCode != 0 {
			err = &types.DecoderErr{
				Code:      int(p.StatusCode),
				Messages:  p.Messages,
				Timestamp: p.Timestamp,
			}
		}
	*/
	return QalcosonicPayload{
		volume: p,
		alarms: ap,
	}, err
}

func Converter(ctx context.Context, deviceID string, payload types.SensorPayload, _ time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(QalcosonicPayload)
	return convert(ctx, deviceID, p)
}

func convert(ctx context.Context, deviceID string, p QalcosonicPayload) ([]lwm2m.Lwm2mObject, error) {

	return convertToLwm2mObjects(ctx, deviceID, p.volume, p.alarms), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p *QalcosonicVolumePayload, ap *QalcosonicAlarmPayload) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	log := logging.GetFromContext(ctx)

	contains := func(strs []string, s string) *bool {
		for _, v := range strs {
			if strings.EqualFold(v, s) {
				b := true
				return &b
			}
		}
		return nil
	}

	if p != nil {
		for _, d := range p.Deltas {
			m3 := d.CumulatedVolume * 0.001

			if d.Timestamp.UTC().After(time.Now().UTC()) {
				log.Warn("time is in the future!", slog.String("device_id", deviceID), slog.String("type_of_meter", p.Type), slog.Time("timestamp", d.Timestamp))
				continue
			}

			wm := lwm2m.NewWaterMeter(deviceID, m3, d.Timestamp)
			wm.TypeOfMeter = &p.Type
			wm.LeakDetected = contains(p.Messages, "Leak")
			wm.BackFlowDetected = contains(p.Messages, "Backflow")

			objects = append(objects, wm)
		}

		if p.Timestamp.UTC().After(time.Now().UTC()) {
			log.Warn("time is in the future!", slog.String("device_id", deviceID), slog.String("type_of_meter", p.Type), slog.Time("timestamp", p.Timestamp))
		} else {
			m3 := p.CurrentVolume * 0.001

			wm := lwm2m.NewWaterMeter(deviceID, m3, p.Timestamp)
			wm.TypeOfMeter = &p.Type
			wm.LeakDetected = contains(p.Messages, "Leak")
			wm.BackFlowDetected = contains(p.Messages, "Backflow")

			objects = append(objects, wm)
		}

		if p.Temperature != nil {
			objects = append(objects, lwm2m.NewTemperature(deviceID, float64(*p.Temperature)/100, p.Timestamp))
		}

		//TODO: create error objects
	}

	if ap != nil {
		log.Warn("unhandled alarm from device", "code", ap.StatusCode, "messages", ap.Messages)
		/*objects = append(objects, lwm2m.Alarm{
			ID_:        deviceID,
			Timestamp_: e.Timestamp,
			AlarmCode:  ap.StatusCode,
			AlarmText:  ap.Messages,
		})*/
	}

	log.Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

func decodePayload(_ context.Context, ue types.Event) (*QalcosonicVolumePayload, *QalcosonicAlarmPayload, error) {
	var err error

	buf := bytes.NewReader(ue.Payload.Data)

	if buf.Len() < 5 && buf.Len() < 42 {
		return nil, nil, errors.New("decoder not implemented or payload to short")
	}

	var p QalcosonicVolumePayload

	switch buf.Len() {
	case 5:
		ap, err := alarmPacketDecoder(buf)
		if err != nil {
			return nil, nil, err
		}
		return nil, &ap, nil
	case 51, 52:
		p, err = w1h(buf)
	case 43, 44, 45, 46, 47:
		p, err = w1e(buf)
	default:
		return nil, nil, fmt.Errorf("unknown payload length %d", buf.Len())
	}

	if err != nil && errors.Is(err, ErrTimeTooFarOff) {
		p, err = w1t(buf)
	}

	if err != nil {
		return nil, nil, err
	}

	return &p, nil, err
}

func alarmPacketDecoder(buf *bytes.Reader) (QalcosonicAlarmPayload, error) {
	var err error
	var epoch uint32
	var statusCode uint8

	p := QalcosonicAlarmPayload{}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err != nil {
		return p, err
	}

	p.Timestamp = time.Unix(int64(epoch), 0).UTC()

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err != nil {
		return p, err
	}

	p.StatusCode = statusCode
	p.Messages = getStatusMessageForAlarmPacket(statusCode)

	return p, nil
}

// Lora Payload (24 hours) “Enhanced”
func w1e(buf *bytes.Reader) (QalcosonicVolumePayload, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var logDateTime uint32
	var volumeAtLogDateTime uint32
	var sensorTime time.Time

	p := QalcosonicVolumePayload{
		Deltas: make([]QalcosonicDeltaVolume, 0),
	}

	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return p, ErrTimeTooFarOff
		}
		p.Timestamp = sensorTime
	} else {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		p.StatusCode = statusCode
		p.Messages = getStatusMessage(statusCode)
	} else {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err != nil {
		return p, err
	}

	p.CurrentVolume = float64(currentVolume)

	var ldt time.Time
	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		dt := time.Unix(int64(logDateTime), 0).UTC()
		if tooFarOff(dt) {
			return p, ErrTimeTooFarOff
		}

		// Log values are always equal to beginning of an hour or a day
		hh, _, _ := dt.Clock()
		y, m, d := dt.Date()
		ldt = time.Date(y, m, d, hh, 0, 0, 0, time.UTC)
	}

	err = binary.Read(buf, binary.LittleEndian, &volumeAtLogDateTime)
	if err != nil {
		return p, err
	}

	p.Deltas = append(p.Deltas, QalcosonicDeltaVolume{
		Timestamp:       ldt,
		CumulatedVolume: float64(volumeAtLogDateTime),
		DeltaVolume:     0.0,
	})

	if d, ok := deltaVolumes(buf, volumeAtLogDateTime, ldt); ok {
		p.Deltas = append(p.Deltas, d...)
	}

	p.Type = "w1e"

	return p, nil
}

func w1t(buf *bytes.Reader) (QalcosonicVolumePayload, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var logDateTime uint32
	var volumeAtLogDateTime uint32

	p := QalcosonicVolumePayload{
		Deltas: make([]QalcosonicDeltaVolume, 0),
	}

	buf.Seek(0, io.SeekStart)

	var sensorTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return p, ErrTimeTooFarOff
		}
		p.Timestamp = sensorTime
	} else {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		p.StatusCode = statusCode
		p.Messages = getStatusMessage(statusCode)
	} else {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &currentVolume)
	if err != nil {
		return p, err
	}

	p.CurrentVolume = float64(currentVolume)

	err = binary.Read(buf, binary.LittleEndian, &temperature)
	if err != nil {
		return p, err
	}

	p.Temperature = &temperature

	var ldt time.Time
	err = binary.Read(buf, binary.LittleEndian, &logDateTime)
	if err == nil {
		// Log values are always equal to beginning of an hour or a day
		dt := time.Unix(int64(logDateTime), 0).UTC()
		hh, _, _ := dt.Clock()
		y, m, d := dt.Date()
		ldt = time.Date(y, m, d, hh, 0, 0, 0, time.UTC)
	} else {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &volumeAtLogDateTime)
	if err != nil {
		return p, err
	}

	p.Deltas = append(p.Deltas, QalcosonicDeltaVolume{
		Timestamp:       ldt,
		CumulatedVolume: float64(volumeAtLogDateTime),
		DeltaVolume:     0.0,
	})

	if d, ok := deltaVolumes(buf, volumeAtLogDateTime, ldt); ok {
		p.Deltas = append(p.Deltas, d...)
	}

	p.Type = "w1t"

	return p, nil
}

func deltaVolumes(buf *bytes.Reader, lastLogValue uint32, logDateTime time.Time) ([]QalcosonicDeltaVolume, bool) {
	var deltaVolume uint16
	deltas := make([]QalcosonicDeltaVolume, 0)

	t := logDateTime
	v := lastLogValue

	for {
		err := binary.Read(buf, binary.LittleEndian, &deltaVolume)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		} else if err != nil {
			return nil, false
		}

		deltas = append(deltas, QalcosonicDeltaVolume{
			Timestamp:       t.Add(time.Hour),
			CumulatedVolume: float64(v + uint32(deltaVolume)),
			DeltaVolume:     float64(deltaVolume),
		})

		t = t.Add(time.Hour)
		v = v + uint32(deltaVolume)
	}

	return deltas, true
}

// Lora Payload (Long) “Extended”
func w1h(buf *bytes.Reader) (QalcosonicVolumePayload, error) {
	var err error

	var frameVersion uint8
	var epoch uint32
	var statusCode uint8
	var logVolumeAtOne uint32

	p := QalcosonicVolumePayload{
		Deltas: make([]QalcosonicDeltaVolume, 0),
	}

	err = binary.Read(buf, binary.LittleEndian, &frameVersion)
	if err == nil {
		p.FrameVersion = frameVersion
	} else {
		return p, err
	}

	var sensorTime time.Time
	var logDateTime time.Time
	err = binary.Read(buf, binary.LittleEndian, &epoch)
	if err == nil {
		sensorTime = time.Unix(int64(epoch), 0).UTC()
		if tooFarOff(sensorTime) {
			return p, ErrTimeTooFarOff
		}
		// First full value is logged at 01:00 time. Other 23 values are differences (increments).
		// All values are logged and always equal to beginning of an hour or a day
		y, m, d := sensorTime.Date()
		logDateTime = time.Date(y, m, d-1, 1, 0, 0, 0, time.UTC)

		p.Timestamp = sensorTime
	} else {
		return p, err
	}

	err = binary.Read(buf, binary.LittleEndian, &statusCode)
	if err == nil {
		p.StatusCode = statusCode
		p.Messages = getStatusMessage(statusCode)
	} else {
		return p, err
	}

	var vol float64 = 0.0
	err = binary.Read(buf, binary.LittleEndian, &logVolumeAtOne)
	if err == nil {
		vol = float64(logVolumeAtOne)
	} else {
		return p, err
	}

	p.Deltas = append(p.Deltas, QalcosonicDeltaVolume{
		Timestamp:       logDateTime,
		CumulatedVolume: vol,
		DeltaVolume:     0.0,
	})

	if d, ok := deltaVolumesH(buf, vol, logDateTime); ok {
		p.Deltas = append(p.Deltas, d...)
	}

	sum := func(v float64) float64 {
		var sum float64 = v
		for _, d := range p.Deltas {
			sum += d.DeltaVolume
		}
		return sum
	}

	p.CurrentVolume = sum(vol)

	p.Type = "w1h"

	return p, nil
}

func deltaVolumesH(buf *bytes.Reader, currentVolume float64, logTime time.Time) ([]QalcosonicDeltaVolume, bool) {
	p := make([]QalcosonicDeltaVolume, 0)

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
		p = append(p, QalcosonicDeltaVolume{
			Timestamp:       deltaTime,
			CumulatedVolume: totalVol,
			DeltaVolume:     float64(deltas[i]),
		})
	}

	return p, len(p) > 0
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
