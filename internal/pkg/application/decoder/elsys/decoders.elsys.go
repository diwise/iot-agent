package elsys

import (
	"context"
	"encoding/json"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type ElsysPayload struct {
	Temperature         *float32 `json:"temperature,omitempty"`
	ExternalTemperature *float32 `json:"externalTemperature,omitempty"`
	Humidity            *int8    `json:"humidity,omitempty"`

	//Acceleration
	X *int8 `json:"x,omitempty"`
	Y *int8 `json:"y,omitempty"`
	Z *int8 `json:"z,omitempty"`

	Light   *uint16 `json:"light,omitempty"`
	Motion  *uint8  `json:"motion,omitempty"`
	CO2     *uint16 `json:"co2,omitempty"`
	VDD     *uint16 `json:"vdd,omitempty"`
	Analog1 *uint16 `json:"analog1,omitempty"`

	//GPS
	Lat *float32 `json:"lat,omitempty"`
	Lon *float32 `json:"long,omitempty"`

	Pulse         *uint16  `json:"pulse1,omitempty"`
	PulseAbs      *uint32  `json:"pulseAbs,omitempty"`
	Pressure      *float32 `json:"pressure,omitempty"`
	Occupancy     *uint8   `json:"occupancy,omitempty"`
	DigitalInput  *bool    `json:"digital,omitempty"`
	DigitalInput2 *bool    `json:"digital2,omitempty"`
	Waterleak     *uint8   `json:"waterleak,omitempty"`
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	p := ElsysPayload{}

	if e.Object == nil {
		p, _ = decodePayload(e.Data)
	} else {
		json.Unmarshal(e.Object, &p)
	}

	objects := []lwm2m.Lwm2mObject{}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.Temperature{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: lwm2m.Round(float64(*p.Temperature)),
		})
	}

	if p.ExternalTemperature != nil {
		objects = append(objects, lwm2m.Temperature{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: lwm2m.Round(float64(*p.ExternalTemperature)),
		})
	}

	if p.Humidity != nil {
		objects = append(objects, lwm2m.Humidity{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: float64(*p.Humidity),
		})
	}

	if p.Light != nil {
		objects = append(objects, lwm2m.Illuminance{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: float64(*p.Light),
		})
	}

	if p.CO2 != nil {
		co2 := float64(*p.CO2)
		objects = append(objects, lwm2m.AirQuality{
			ID_:        deviceID,
			Timestamp_: e.Timestamp,
			CO2:        &co2,
		})
	}

	if p.VDD != nil {
		vdd := float64(*p.VDD) / 1000
		objects = append(objects, lwm2m.Battery{
			ID_:            deviceID,
			Timestamp_:     e.Timestamp,
			BatteryVoltage: &vdd,
		})
	}

	if p.Occupancy != nil {
		objects = append(objects, lwm2m.Presence{
			ID_:               deviceID,
			Timestamp_:        e.Timestamp,
			DigitalInputState: *p.Occupancy == 2,
		})
	}

	if p.DigitalInput != nil {
		var pulseAbs *int
		if p.PulseAbs != nil {
			pulseAbs = new(int)
			*pulseAbs = int(*p.PulseAbs)
		}
		objects = append(objects, lwm2m.DigitalInput{
			ID_:                 deviceID,
			Timestamp_:          e.Timestamp,
			DigitalInputState:   *p.DigitalInput,
			DigitalInputCounter: pulseAbs,
		})
	}

	return objects, nil
}
