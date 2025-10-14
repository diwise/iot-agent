package qalcosonic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

var ErrTimeTooFarOff = fmt.Errorf("sensor time is too far off in the future")

const NoError = 0x00

type WaterMeterReading struct {
	Current      float64   `json:"current_volume,omitempty"`
	LastLogValue uint32    `json:"last_log_value,omitempty"`
	Volumes      []Value   `json:"volumes,omitempty"`
	FrameVersion uint8     `json:"frame_version,omitempty"`
	Messages     []string  `json:"messages,omitempty"`
	StatusCode   uint8     `json:"status_code,omitempty"`
	Temperature  *uint16   `json:"temperature,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type,omitempty"`
}

type Alarm struct {
	Timestamp  time.Time `json:"timestamp"`
	StatusCode uint8     `json:"status_code,omitempty"`
	Messages   []string  `json:"messages,omitempty"`
}

type Payload struct {
	Reading *WaterMeterReading `json:"volume,omitempty"`
	Alarms  *Alarm             `json:"alarms,omitempty"`
}

func (a Payload) BatteryLevel() *int {
	return nil
}
func (a Payload) Error() (string, []string) {
	if len(a.Reading.Messages) > 0 {
		m := []string{}
		for _, v := range a.Reading.Messages {
			if v != "No error" {
				m = append(m, v)
			}
		}

		if a.Reading.StatusCode == NoError {
			return "0", m
		}

		return strconv.Itoa(int(a.Reading.StatusCode)), m
	}
	return "0", []string{}
}

type Value struct {
	Volume    float64
	Delta     float64
	Timestamp time.Time
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	var err error

	if e.Payload.FPort != 100 {
		return Payload{}, fmt.Errorf("unsupported fPort %d", e.Payload.FPort)
	}

	p, ap, err := decode(ctx, e)
	if err != nil {
		return nil, err
	}

	return Payload{
		Reading: p,
		Alarms:  ap,
	}, err
}

func DecoderW1h(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	if e.Payload.FPort != 100 {
		return Payload{}, fmt.Errorf("unsupported fPort %d", e.Payload.FPort)
	}

	buf := bytes.NewReader(e.Payload.Data)

	if buf.Len() != 51 && buf.Len() != 52 {
		return Payload{}, fmt.Errorf("unsupported payload length %d for w1h decoder", buf.Len())
	}

	p, err := w1h(buf)
	if err != nil {
		return nil, err
	}

	return Payload{
		Reading: &p,
	}, nil
}

func DecoderW1e(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	if e.Payload.FPort != 100 {
		return Payload{}, fmt.Errorf("unsupported fPort %d", e.Payload.FPort)
	}

	buf := bytes.NewReader(e.Payload.Data)

	if buf.Len() != 43 && buf.Len() != 44 && buf.Len() != 45 && buf.Len() != 46 && buf.Len() != 47 {
		return Payload{}, fmt.Errorf("unsupported payload length %d for w1e decoder", buf.Len())
	}

	p, err := w1e(buf)
	if err != nil {
		return nil, err
	}

	return Payload{
		Reading: &p,
	}, nil
}

func DecoderW1t(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	if e.Payload.FPort != 100 {
		return Payload{}, fmt.Errorf("unsupported fPort %d", e.Payload.FPort)
	}

	buf := bytes.NewReader(e.Payload.Data)

	switch buf.Len() {
	case 5:
		ap, err := alarmPacketDecoder(buf)
		if err != nil {
			return nil, err
		}
		return Payload{
			Alarms: &ap,
		}, nil
	case 47:
		p, err := w1t(buf)
		if err != nil {
			return nil, err
		}
		return Payload{
			Reading: &p,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported payload length %d for w1t decoder", buf.Len())
	}
}

func Converter(ctx context.Context, deviceID string, payload types.SensorPayload, _ time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(Payload)
	return convert(ctx, deviceID, p)
}

func convert(ctx context.Context, deviceID string, p Payload) ([]lwm2m.Lwm2mObject, error) {
	log := logging.GetFromContext(ctx)

	objects := convertToLwm2mObjects(ctx, deviceID, p.Reading, p.Alarms)

	prev := time.Time{}

	for _, obj := range objects {
		if prev.After(obj.Timestamp()) {
			log.Warn("out of order timestamps", slog.String("device_id", deviceID), slog.String("object_id", obj.ObjectID()), slog.Time("prev", prev), slog.Time("current", obj.Timestamp()))
			return nil, fmt.Errorf("out of order timestamps")
		}

		prev = obj.Timestamp()
	}

	return objects, nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p *WaterMeterReading, ap *Alarm) []lwm2m.Lwm2mObject {
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
		for _, d := range p.Volumes {
			if d.Timestamp.UTC().After(time.Now().UTC()) {
				log.Warn("time is in the future!", slog.String("device_id", deviceID), slog.String("type_of_meter", p.Type), slog.Time("timestamp", d.Timestamp))
				continue
			}

			m3 := d.Volume * 0.001

			wm := lwm2m.NewWaterMeter(deviceID, m3, d.Timestamp)
			wm.TypeOfMeter = &p.Type
			wm.LeakDetected = contains(p.Messages, "Leak")
			wm.BackFlowDetected = contains(p.Messages, "Backflow")

			objects = append(objects, wm)
		}

		if p.Timestamp.UTC().After(time.Now().UTC()) {
			log.Warn("time is in the future!", slog.String("device_id", deviceID), slog.String("type_of_meter", p.Type), slog.Time("timestamp", p.Timestamp))
		}

		if p.Temperature != nil {
			objects = append(objects, lwm2m.NewTemperature(deviceID, float64(*p.Temperature)/100, p.Timestamp))
		}
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

func decode(_ context.Context, ue types.Event) (*WaterMeterReading, *Alarm, error) {
	var err error

	buf := bytes.NewReader(ue.Payload.Data)

	if buf.Len() < 5 && buf.Len() < 42 {
		return nil, nil, errors.New("decoder not implemented or payload to short")
	}

	var p WaterMeterReading

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

func alarmPacketDecoder(buf *bytes.Reader) (Alarm, error) {
	var err error
	var epoch uint32
	var statusCode uint8

	p := Alarm{}

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

// Lora Payload (Long) “Extended”
func w1e(buf *bytes.Reader) (WaterMeterReading, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var logDateTime uint32
	var volumeAtLogDateTime uint32
	var sensorTime time.Time

	p := WaterMeterReading{
		Volumes: make([]Value, 0),
	}

	buf.Seek(0, io.SeekStart)

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

	p.Current = float64(currentVolume)

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

	if sensorTime.Sub(ldt) > 24*time.Hour || sensorTime.Sub(ldt) < -48*time.Hour {
		return p, ErrTimeTooFarOff
	}

	err = binary.Read(buf, binary.LittleEndian, &volumeAtLogDateTime)
	if err != nil {
		return p, err
	}

	p.LastLogValue = volumeAtLogDateTime

	p.Volumes = append(p.Volumes, Value{
		Timestamp: ldt,
		Volume:    float64(volumeAtLogDateTime),
		Delta:     0.0,
	})

	if d, ok := decodeDeltaVolumesExtended(buf, volumeAtLogDateTime, ldt); ok {
		p.Volumes = append(p.Volumes, d...)
	}

	p.Type = "w1e"

	return p, nil
}

// Lora Payload (Long) “Extended” with Temperature
func w1t(buf *bytes.Reader) (WaterMeterReading, error) {
	var err error

	var epoch uint32
	var statusCode uint8
	var currentVolume uint32
	var temperature uint16
	var logDateTime uint32
	var volumeAtLogDateTime uint32

	p := WaterMeterReading{
		Volumes: make([]Value, 0),
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

	p.Current = float64(currentVolume)

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

	p.LastLogValue = volumeAtLogDateTime

	p.Volumes = append(p.Volumes, Value{
		Timestamp: ldt,
		Volume:    float64(volumeAtLogDateTime),
		Delta:     0.0,
	})

	if d, ok := decodeDeltaVolumesExtended(buf, volumeAtLogDateTime, ldt); ok {
		p.Volumes = append(p.Volumes, d...)
	}

	p.Type = "w1t"

	return p, nil
}

func decodeDeltaVolumesExtended(buf *bytes.Reader, lastLogValue uint32, logDateTime time.Time) ([]Value, bool) {
	var deltaVolume uint16
	deltas := make([]Value, 0)

	t := logDateTime
	v := lastLogValue

	for {
		err := binary.Read(buf, binary.LittleEndian, &deltaVolume)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		} else if err != nil {
			return nil, false
		}

		deltas = append(deltas, Value{
			Timestamp: t.Add(time.Hour),
			Volume:    float64(v + uint32(deltaVolume)),
			Delta:     float64(deltaVolume),
		})

		t = t.Add(time.Hour)
		v = v + uint32(deltaVolume)
	}

	return deltas, true
}

// Lora Payload (Long) "Enhanced"
func w1h(buf *bytes.Reader) (WaterMeterReading, error) {
	var err error

	var frameVersion uint8
	var epoch uint32
	var statusCode uint8
	var logVolumeAtOne uint32

	p := WaterMeterReading{
		Volumes: make([]Value, 0),
	}

	buf.Seek(0, io.SeekStart)

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

	p.LastLogValue = logVolumeAtOne

	p.Volumes = append(p.Volumes, Value{
		Timestamp: logDateTime,
		Volume:    vol,
		Delta:     0.0,
	})

	if d, ok := decodeDeltaVolumesEnhanced(buf, vol, logDateTime); ok {
		p.Volumes = append(p.Volumes, d...)
	}

	sum := func(v float64) float64 {
		var sum float64 = v
		for _, d := range p.Volumes {
			sum += d.Delta
		}
		return sum
	}

	p.Current = sum(vol)

	p.Type = "w1h"

	return p, nil
}

func decodeDeltaVolumesEnhanced(buf *bytes.Reader, currentVolume float64, logTime time.Time) ([]Value, bool) {
	p := make([]Value, 0)

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

	for i := range 23 {
		totalVol += float64(deltas[i])
		deltaTime = deltaTime.Add(1 * time.Hour)

		p = append(p, Value{
			Timestamp: deltaTime,
			Volume:    totalVol,
			Delta:     float64(deltas[i]),
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
