package enviot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/lwm2m"
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

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	obj := EnviotPayload{}

	err := json.Unmarshal(e.Object, &obj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal enviot payload: %s", err.Error())
	}

	objects := []lwm2m.Lwm2mObject{}

	if obj.Payload.Temperature != nil {
		objects = append(objects, lwm2m.Temperature{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: lwm2m.Round(float64(*obj.Payload.Temperature)),
		})
	}

	if obj.Payload.Humidity != nil {
		objects = append(objects, lwm2m.Humidity{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: float64(*obj.Payload.Humidity),
		})
	}

	if obj.Payload.Battery != nil {
		objects = append(objects, lwm2m.Battery{
			ID_:          deviceID,
			Timestamp_:   e.Timestamp,
			BatteryLevel: *obj.Payload.Battery,
		})
	}

	if obj.Payload.SensorStatus == 0 && obj.Payload.SnowHeight != nil {
		applicationType := "SnowHeight"
		objects = append(objects, lwm2m.Distance{
			ID_:             deviceID,
			Timestamp_:      e.Timestamp,
			SensorValue:     float64(*obj.Payload.SnowHeight),
			ApplicationType: &applicationType,
		})
	}

	return objects, nil
}
