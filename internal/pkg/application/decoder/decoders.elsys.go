package decoder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func ElsysDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	d := struct {
		Temperature         *float32 `json:"temperature,omitempty"`
		ExternalTemperature *float32 `json:"externalTemperature,omitempty"`
		Vdd                 *int     `json:"vdd,omitempty"`
		CO2                 *int     `json:"co2,omitempty"`
		Humidity            *int     `json:"humidity,omitempty"`
		Light               *int     `json:"lights,omitempty"`
		Motion              *int     `json:"motion,omitempty"`
	}{}

	err := json.Unmarshal(ue.Object, &d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal elsys payload: %s", err.Error())
	}

	var decorators []payload.PayloadDecoratorFunc

	if d.Temperature != nil {
		decorators = append(decorators, payload.Temperature(*d.Temperature))
	}

	if d.ExternalTemperature != nil {
		decorators = append(decorators, payload.Temperature(*d.ExternalTemperature))
	}

	if d.CO2 != nil {
		decorators = append(decorators, payload.CO2(*d.CO2))
	}

	if d.Humidity != nil {
		decorators = append(decorators, payload.Humidity(*d.Humidity))
	}

	if d.Light != nil {
		decorators = append(decorators, payload.Light(*d.Light))
	}

	if d.Motion != nil {
		decorators = append(decorators, payload.Motion(*d.Motion))
	}

	if d.Vdd != nil {
		decorators = append(decorators, payload.BatteryLevel(*d.Vdd))
	}

	if p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...); err == nil {
		err := fn(ctx, p)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}
