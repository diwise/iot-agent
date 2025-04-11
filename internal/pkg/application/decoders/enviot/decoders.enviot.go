package enviot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type EnviotPayload struct {
	Payload struct {
		Battery      *int     `json:"battery,omitempty"`
		Humidity     *float32 `json:"humidity,omitempty"`
		SensorStatus int      `json:"sensorStatus"`
		SnowHeight   *int     `json:"snowHeight,omitempty"`
		Temperature  *float32 `json:"temperature,omitempty"`
	} `json:"payload"`
}

func (a EnviotPayload) BatteryLevel() *int {
	if a.Payload.Battery != nil {
		bat := int(*a.Payload.Battery)
		return &bat
	}

	return nil
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	obj := EnviotPayload{}

	err := json.Unmarshal(e.Payload.Object, &obj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal enviot payload: %s", err.Error())
	}

	return obj, nil
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(EnviotPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p EnviotPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Payload.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, float64(*p.Payload.Temperature), ts))
	}

	if p.Payload.Humidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, float64(*p.Payload.Humidity), ts))
	}

	if p.Payload.Battery != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bl := int(*p.Payload.Battery)
		d.BatteryLevel = &bl
		objects = append(objects, d)
	}

	if p.Payload.SensorStatus == 0 && p.Payload.SnowHeight != nil {
		applicationType := "SnowHeight"
		d := lwm2m.NewDistance(deviceID, float64(*p.Payload.SnowHeight), ts)
		d.ApplicationType = &applicationType
		objects = append(objects, d)
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}
