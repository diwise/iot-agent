package sensative

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type SensativePayload struct {
	BatteryLevel *int
	Temperature  *float64
	Humidity     *float32
	DoorReport   *bool
	DoorAlarm    *bool
	Presence     *bool
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	if len(e.Data) < 2 {
		return nil, errors.New("payload too short")
	}

	p, err := decodeSensativeMeasurements(e.Data)
	if err != nil {
		return nil, err
	}

	objects := convertToLwm2mObjects(ctx, deviceID, p, e.Timestamp)

	if len(objects) == 0 {
		checkIn := struct {
			BuildID struct {
				ID       int  `json:"id"`
				Modified bool `json:"modified"`
			} `json:"buildId"`
			HistorySeqNr   uint16 `json:"historySeqNr"`
			PrevHistorySeq uint16 `json:"prevHistSeqNr"`
		}{}

		err = json.Unmarshal(e.Object, &checkIn)
		if err != nil {
			return nil, err
		}

		objects = append(objects, lwm2m.NewDevice(deviceID, e.Timestamp))
	}

	return objects, nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p SensativePayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := make([]lwm2m.Lwm2mObject, 0)

	if p.BatteryLevel != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bat := int(*p.BatteryLevel)
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

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

func decodeSensativeMeasurements(b []byte) (SensativePayload, error) {
	p := SensativePayload{}

	pos := 2

	for pos < len(b) {
		channel := b[pos] & 0x7F
		pos = pos + 1
		size := 1

		switch channel {
		case 1: // battery
			bl := int(b[pos])
			p.BatteryLevel = &bl
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
