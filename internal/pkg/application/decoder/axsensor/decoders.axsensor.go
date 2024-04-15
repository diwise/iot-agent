package axsensor

import (
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type AxsensorPayload struct {
	Distance         *float64 `json:"level,omitempty"`
	Pressure         *float64 `json:"pressure,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	RelativeHumidity *float64 `json:"relativeHumidity,omitempty"`
	Vbat             *float64 `json:"vbat,omitempty"`
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	//   "payload": "804f21" har längd 3 och blir till error
	// if len(e.Data)%2 != 0 {
	// 	return nil, errors.New("not valid payload")
	// }

	if e.FPort != 2 {
		return nil, errors.New("not valid fPort")
	}

	p, err := decode(e.Data)
	if err != nil {
		return nil, err
	}

	return convertToLwm2mObjects(deviceID, p, e.Timestamp), nil
}

func decode(b []byte) (AxsensorPayload, error) {
	p := AxsensorPayload{}
	slen := len(b)
	idx := 0
	blen := slen / 2

	for idx < blen {
		switch b[idx] {
		case 0x80:
			level := 1400 - (binary.LittleEndian.Uint16(b[idx+1:idx+3]) / 10) //472
			sewerdistance := float64(level)
			p.Distance = &sewerdistance
		case 0xA1:
			pressure := (float64(b[idx+1]) + float64(b[idx+2])*256) * 100 //Pa
			p.Pressure = &pressure
		case 0xA2:
			temperature := (binary.LittleEndian.Uint16(b[idx+1 : idx+3])) //C 5.6
			temperatureLevel := float64(temperature) / 10
			p.Temperature = &temperatureLevel
		case 0xA3:
			humidity := (float64(b[idx+1]) + float64(b[idx+2])*256) / 1024 * 100 //rh%
			p.RelativeHumidity = &humidity
		case 0xA4:
			vbat := binary.LittleEndian.Uint16(b[idx+1 : idx+3]) //mV 3488
			batteryLevel := float64(vbat)
			p.Vbat = &batteryLevel
		}
		switch {
		case b[idx] < 0x40:
			idx++
		case b[idx] < 0x80:
			idx += 2
		case b[idx] < 0xC0:
			idx += 3
		default:
			idx += 5
		}
	}
	return p, nil

}

func convertToLwm2mObjects(deviceID string, p AxsensorPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Distance != nil {
		objects = append(objects, lwm2m.NewDistance(deviceID, float64(*p.Distance), ts))
	}

	if p.Pressure != nil {
		objects = append(objects, lwm2m.NewPressure(deviceID, float64(*p.Pressure), ts))
	}

	if p.RelativeHumidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, float64(*p.RelativeHumidity), ts))
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, float64(*p.Temperature), ts))
	}

	if p.Vbat != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bat := int(*p.Vbat)
		d.PowerSourceVoltage = &bat
		objects = append(objects, d)
	}

	return objects
}
