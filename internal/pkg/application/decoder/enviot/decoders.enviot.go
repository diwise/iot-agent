package enviot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func EnviotDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	obj := struct {
		Payload struct {
			Battery      *int     `json:"battery,omitempty"`
			Humidity     *int     `json:"humidity,omitempty"`
			SensorStatus int      `json:"sensorStatus"`
			SnowHeight   *int     `json:"snowHeight,omitempty"`
			Temperature  *float32 `json:"temperature,omitempty"`
		} `json:"payload"`
	}{}

	err := json.Unmarshal(ue.Object, &obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal enviot payload: %s", err.Error())
	}

	var decorators []payload.PayloadDecoratorFunc

	if obj.Payload.Temperature != nil {
		decorators = append(decorators, payload.Temperature(float64(*obj.Payload.Temperature)))
	}

	if obj.Payload.Battery != nil {
		decorators = append(decorators, payload.BatteryLevel(*obj.Payload.Battery))
	}

	if obj.Payload.Humidity != nil {
		decorators = append(decorators, payload.Humidity(*obj.Payload.Humidity))
	}

	if obj.Payload.SensorStatus == 0 && obj.Payload.SnowHeight != nil {
		decorators = append(decorators, payload.SnowHeight(*obj.Payload.SnowHeight))
	}

	decorators = append(decorators, payload.Status(uint8(obj.Payload.SensorStatus), nil))

	p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...)
	if err != nil {
		return err
	}

	return fn(ctx, p)
}
