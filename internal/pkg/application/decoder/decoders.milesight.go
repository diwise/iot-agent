package decoder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func MilesightDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	d := struct {
		Temperature *float32 `json:"temperature,omitempty"`
		Humidity    *float32 `json:"humidity,omitempty"`
		CO2         *int     `json:"co2,omitempty"`
		Battery     *int     `json:"battery,omitempty"`
	}{}

	err := json.Unmarshal(ue.Object, &d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal milesight payload: %s", err.Error())
	}

	var decorators []payload.PayloadDecoratorFunc

	if d.Temperature != nil {
		decorators = append(decorators, payload.Temperature(float64(*d.Temperature)))
	}

	if d.Humidity != nil {
		decorators = append(decorators, payload.Humidity(int(*d.Humidity)))
	}

	if d.CO2 != nil {
		decorators = append(decorators, payload.CO2(*d.CO2))
	}

	if d.Battery != nil {
		decorators = append(decorators, payload.BatteryLevel(*d.Battery))
	}

	if p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...); err == nil {
		return fn(ctx, p)
	} else {
		return err
	}
}
