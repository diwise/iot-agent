package milesight

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type MilesightPayload struct {
	Battery     *int
	CO2         *int
	Distance    *float64
	Humidity    *float64
	Temperature *float64
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	p, err := decode(e.Data)
	if err != nil {
		return nil, err
	}

	return convertToLwm2mObjects(deviceID, p, e.Timestamp), nil
}

func convertToLwm2mObjects(deviceID string, p MilesightPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Battery != nil {
		objects = append(objects, lwm2m.NewBattery(deviceID, *p.Battery, ts))
	}

	if p.CO2 != nil {
		co2 := float64(*p.CO2)
		objects = append(objects, lwm2m.NewAirQuality(deviceID, co2, ts))
	}

	if p.Distance != nil {
		objects = append(objects, lwm2m.NewDistance(deviceID, *p.Distance, ts))
	}

	if p.Humidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, *p.Humidity, ts))
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, *p.Temperature, ts))
	}

	return objects
}

func decode(b []byte) (MilesightPayload, error) {
	p := MilesightPayload{}

	numberOfBytes := len(b)

	const (
		Battery     uint16 = 373  // 0x0175
		CO2         uint16 = 1917 // 0x077D
		Distance    uint16 = 898  // 0x0382
		Humidity    uint16 = 1128 // 0x0468
		Temperature uint16 = 871  // 0x0367
	)

	data_length := map[uint16]int{
		Battery: 1, CO2: 2, Distance: 2, Humidity: 1, Temperature: 2,
	}

	rangeCheck := func(atPos, numBytes int) bool {
		return (atPos + numBytes - 1) < numberOfBytes
	}

	pos := 0
	size := 0
	const HeaderSize int = 2

	for pos < numberOfBytes {

		if !rangeCheck(pos, HeaderSize) {
			return p, errors.New("range check failed before trying to read channel header")
		}

		channel_header := binary.BigEndian.Uint16(b[pos : pos+HeaderSize])
		pos = pos + HeaderSize

		var ok bool
		if size, ok = data_length[channel_header]; !ok {
			return p, fmt.Errorf("unknown channel header %X", channel_header)
		}

		if !rangeCheck(pos, size) {
			return p, errors.New("range check failed before trying to read channel value")
		}

		switch channel_header {
		case Battery:
			b := int(b[pos])
			p.Battery = &b
		case CO2:
			co2 := int(binary.LittleEndian.Uint16(b[pos : pos+2]))
			p.CO2 = &co2
		case Distance:
			millimeters := float64(binary.LittleEndian.Uint16(b[pos : pos+2]))
			meters := millimeters / 1000.0
			// convert distance to meters
			p.Distance = &meters
		case Humidity:
			h := float64(b[pos]) / 2.0
			p.Humidity = &h
		case Temperature:
			t := float64(binary.LittleEndian.Uint16(b[pos:pos+2])) / 10.0
			p.Temperature = &t
		default:
			return p, fmt.Errorf("unknown channel header %X", channel_header)
		}

		pos = pos + size
	}

	return p, nil
}
