package sensative

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type SensativePayload struct {
	BatteryLevel_ *int
	Temperature  *float64
	Humidity     *float32
	DoorReport   *bool
	DoorAlarm    *bool
	Presence     *bool
	CheckIn      *bool
}

func (a SensativePayload) BatteryLevel() *int {
	return a.BatteryLevel_
}


func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	if e.Payload == nil {
		return nil, types.ErrPayloadContainsNoData
	}

	if e.Payload.FPort == 2 {
		t := true
		return SensativePayload{
			CheckIn: &t,
		}, nil
	}

	p, err := decode(e.Payload.Data)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(SensativePayload)
	objects := convertToLwm2mObjects(ctx, deviceID, p, ts)

	return objects, nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p SensativePayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := make([]lwm2m.Lwm2mObject, 0)

	if p.BatteryLevel_ != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bat := int(*p.BatteryLevel_)
		d.BatteryLevel = &bat
		objects = append(objects, d)
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, *p.Temperature, ts))
	}

	if p.Humidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, float64(*p.Humidity), ts))
	}
	/*
		if p.DoorReport != nil {
			objects = append(objects, lwm2m.DigitalInput{
				ID_:               deviceID,
				Timestamp_:        e.Timestamp,
				DigitalInputState: *p.DoorReport,
			})
		}

		if p.DoorAlarm != nil {
			objects = append(objects, lwm2m.DigitalInput{
				ID_:               deviceID,
				Timestamp_:        e.Timestamp,
				DigitalInputState: *p.DoorAlarm,
			})
		}
	*/
	if p.Presence != nil {
		objects = append(objects, lwm2m.NewPresence(deviceID, *p.Presence, ts))
	}

	if p.CheckIn != nil && *p.CheckIn {
		objects = append(objects, lwm2m.NewDevice(deviceID, ts))
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

func decode(b []byte) (SensativePayload, error) {
	p := SensativePayload{}

	pos := 2

	for pos < len(b) {
		channel := b[pos] & 0x7F
		pos = pos + 1
		size := 1

		switch channel {
		case 1: // battery
			bl := int(b[pos])
			p.BatteryLevel_ = &bl
		case 2: // temp report
			size = 2
			t := float64(binary.BigEndian.Uint16(b[pos:pos+2]) / 10)
			p.Temperature = &t
			// TODO: Handle sub zero readings
		case 4: // average temp report
			size = 2
		case 6: // humidity report
			h := float32(b[pos]) / 2.0
			p.Humidity = &h
		case 7: // lux report
			size = 2
		case 8: // lux2 report
			size = 2
		case 9: // door report
			dr := b[pos] != 0
			p.DoorReport = &dr
		case 10: // door alarm
			da := b[pos] != 0
			p.DoorAlarm = &da
		case 21: // close proximity alarm
			pr := b[pos] != 0
			p.Presence = &pr
		case 110: // check in confirmed
			size = 8
		default:
			fmt.Printf("unknown channel %d\n", channel)
			size = 20
		}

		pos = pos + size
	}

	return p, nil
}
