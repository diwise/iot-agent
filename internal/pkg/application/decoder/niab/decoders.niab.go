package niab

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
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

	return convertToLwm2mObjects(deviceID, p, e.Timestamp), nil
}

func convertToLwm2mObjects(deviceID string, p NiabPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	d := lwm2m.NewDevice(deviceID, ts)
	bat := int(p.Battery)
	d.BatteryLevel = &bat	
	objects = append(objects, d)

	objects = append(objects, lwm2m.NewTemperature(deviceID, p.Temperature, ts))

	if p.Distance != nil {
		objects = append(objects, lwm2m.NewDistance(deviceID, *p.Distance, ts))
	}

	return objects
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
