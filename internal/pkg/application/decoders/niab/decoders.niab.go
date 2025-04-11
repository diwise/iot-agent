package niab

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type NiabPayload struct {
	Battery     int
	Temperature float64
	Distance    *float64
}

func (a NiabPayload) BatteryLevel() *int {
	return &a.Battery
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	return decode(e.Payload.Object)
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(NiabPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p NiabPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	d := lwm2m.NewDevice(deviceID, ts)
	bat := int(p.Battery)
	d.BatteryLevel = &bat
	objects = append(objects, d)

	objects = append(objects, lwm2m.NewTemperature(deviceID, p.Temperature, ts))

	if p.Distance != nil {
		objects = append(objects, lwm2m.NewDistance(deviceID, *p.Distance, ts))
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

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
