package niab

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/lwm2m"
)

type NiabPayload struct {
	Battery     int
	Temperature float64
	Distance    *float64
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	p, err := decode(e.Object)
	if err != nil {
		return nil, err
	}

	objects := []lwm2m.Lwm2mObject{}

	objects = append(objects, lwm2m.Battery{
		ID_:          deviceID,
		Timestamp_:   e.Timestamp,
		BatteryLevel: p.Battery,
	})

	objects = append(objects, lwm2m.Temperature{
		ID_:         deviceID,
		Timestamp_:  e.Timestamp,
		SensorValue: lwm2m.Round(p.Temperature),
	})

	if p.Distance != nil {
		objects = append(objects, lwm2m.Distance{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: *p.Distance,
		})
	}

	return objects, nil
}

func decode(b []byte) (NiabPayload, error) {
	p := NiabPayload{}

	if len(b) < 4 {
		return p, errors.New("payload too short")
	} else if len(b) > 4 {
		return p, errors.New("payload too long")
	}

	bat := int(b[0])
	temp := int(b[1])

	if temp > 127 {
		temp = temp - 255
	}

	p.Battery = bat * 100 / 255
	p.Temperature = float64(temp)

	var distance int16
	binary.Read(bytes.NewReader(b[2:4]), binary.BigEndian, &distance)

	// filter out sensor reading errors
	// TODO: Figure out if we want to signal this error in some way
	if distance != -1 {
		// convert the reported distance in millimeters to meters instead
		dist := float64(distance) / 1000.0
		p.Distance = &dist
	} else {
		return p, errors.New("sensor reading error")
	}

	return p, nil
}
