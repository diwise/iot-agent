package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
)

func ElsysDecoder(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	pp := &Payload{
		DevEUI:     ue.DevEui,
		SensorType: ue.SensorType,
		Timestamp:  ue.Timestamp.Format(time.RFC3339Nano),
	}

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

	if d.Temperature != nil {
		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			*d.Temperature,
		}
		pp.Measurements = append(pp.Measurements, temp)
	}

	if d.ExternalTemperature != nil {
		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			*d.ExternalTemperature,
		}
		pp.Measurements = append(pp.Measurements, temp)
	}

	if d.CO2 != nil {
		co2 := struct {
			CO2 int `json:"co2"`
		}{
			*d.CO2,
		}
		pp.Measurements = append(pp.Measurements, co2)
	}

	if d.Humidity != nil {
		hmd := struct {
			Humidity int `json:"humidity"`
		}{
			*d.Humidity,
		}
		pp.Measurements = append(pp.Measurements, hmd)
	}

	if d.Light != nil {
		lght := struct {
			Light int `json:"light"`
		}{
			*d.Light,
		}
		pp.Measurements = append(pp.Measurements, lght)
	}

	if d.Motion != nil {
		mtn := struct {
			Motion int `json:"motion"`
		}{
			*d.Motion,
		}
		pp.Measurements = append(pp.Measurements, mtn)
	}

	if d.Vdd != nil {
		bat := struct {
			BatteryLevel int `json:"battery_level"`
		}{
			*d.Vdd, // TODO: Adjust for max VDD
		}
		pp.BatteryLevel = bat.BatteryLevel
		pp.Measurements = append(pp.Measurements, bat)
	}

	err = fn(ctx, *pp)
	if err != nil {
		return err
	}

	return nil
}
