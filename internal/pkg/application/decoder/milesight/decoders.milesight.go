package milesight

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func Decoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {

	var decorators []payload.PayloadDecoratorFunc

	decoratorAppender := func(m payload.PayloadDecoratorFunc) {
		decorators = append(decorators, m)
	}

	if err := decodeMilesightMeasurements(ue.Data, decoratorAppender); err != nil {
		return err
	}

	p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...)
	if err != nil {
		return err
	}

	return fn(ctx, p)
}

func decodeMilesightMeasurements(b []byte, callback func(m payload.PayloadDecoratorFunc)) error {
	numberOfBytes := len(b)

	const (
		Battery       uint16 = 373  // 0x0175
		CO2           uint16 = 1917 // 0x077D
		Distance      uint16 = 898  // 0x0382
		Humidity      uint16 = 1128 // 0x0468
		Temperature   uint16 = 871  // 0x0367
		DistanceEM400 uint16 = 1154 // 0x482
		Position      uint16 = 1280 //0x500
	)

	data_length := map[uint16]int{
		Battery: 1, CO2: 2, Distance: 2, Humidity: 1, Temperature: 2, DistanceEM400: 2, Position: 1,
	}

	rangeCheck := func(atPos, numBytes int) bool {
		return (atPos + numBytes - 1) < numberOfBytes
	}

	pos := 0
	size := 0

	for pos < numberOfBytes {

		const HeaderSize int = 2
		if !rangeCheck(pos, HeaderSize) {
			return errors.New("range check failed before trying to read channel header")
		}

		channel_header := binary.BigEndian.Uint16(b[pos : pos+HeaderSize])
		pos = pos + HeaderSize

		var ok bool
		if size, ok = data_length[channel_header]; !ok {
			return fmt.Errorf("unknown channel header %X", channel_header)
		}

		if !rangeCheck(pos, size) {
			return errors.New("range check failed before trying to read channel value")
		}

		switch channel_header {
		case Battery:
			callback(payload.BatteryLevel(int(b[pos])))
		case CO2:
			callback(payload.CO2(int(binary.LittleEndian.Uint16(b[pos : pos+2]))))
		case Distance:
			millimeters := float64(binary.LittleEndian.Uint16(b[pos : pos+2]))
			// convert distance to meters
			callback(payload.Distance(millimeters / 1000.0))
		case Humidity:
			callback(payload.Humidity(float32(b[pos]) / 2.0))
		case Temperature:
			callback(payload.Temperature(float64(binary.LittleEndian.Uint16(b[pos:pos+2])) / 10.0))
		case DistanceEM400:
			millimeters := float64(binary.LittleEndian.Uint16(b[pos : pos+2]))
			// convert distance to meters
			callback(payload.Distance(millimeters / 1000.0))
		case Position:
			p := "normal"
			if float32(b[pos]) == 1 {
				p = "tilt"
			}
			callback(payload.Position(p))
		default:
			return fmt.Errorf("unknown channel header %X", channel_header)
		}

		pos = pos + size
	}

	return nil
}
