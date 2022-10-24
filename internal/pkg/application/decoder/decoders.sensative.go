package decoder

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func SensativeDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {

	if len(ue.Data) < 2 {
		return errors.New("payload too short")
	}

	var decorators []payload.PayloadDecoratorFunc

	err := decodeSensativeMeasurements(ue.Data, func(m payload.PayloadDecoratorFunc) {
		decorators = append(decorators, m)
	})
	if err != nil {
		return err
	}

	p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...)
	if err != nil {
		return err
	}

	return fn(ctx, p)
}

func decodeSensativeMeasurements(b []byte, callback func(m payload.PayloadDecoratorFunc)) error {
	pos := 2

	for pos < len(b) {
		channel := b[pos] & 0x7F
		pos = pos + 1
		size := 1

		switch channel {
		case 1: // battery
			callback(payload.BatteryLevel(int(b[pos])))
		case 2: // temp report
			size = 2
			// TODO: Handle sub zero readings
			callback(payload.Temperature(float32(binary.BigEndian.Uint16(b[pos:pos+2]) / 10)))
		case 4: // average temp report
			size = 2
		case 6: // humidity report
			callback(payload.Humidity(int(b[pos] / 2)))
		case 7: // lux report
			size = 2
		case 8: // lux2 report
			size = 2
		case 9: // door report
			callback(payload.DoorReport(b[pos] != 0))
		case 10: // door alarm
			callback(payload.DoorAlarm(b[pos] != 0))
		default:
			fmt.Printf("unknown channel %d\n", channel)
			size = 20
		}

		pos = pos + size
	}

	return nil
}

type Measurement interface{}
