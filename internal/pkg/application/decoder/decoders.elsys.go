package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func ElsysDecoder(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {

	d := struct {
		DevEUI     string `json:"devEUI"`
		FPort      int    `json:"fPort"`
		SensorType string `json:"deviceProfileName"`
		Data       string `json:"data"`
		Object     struct {
			Temperature         *float32 `json:"temperature,omitempty"`
			ExternalTemperature *float32 `json:"externalTemperature,omitempty"`
			Vdd                 *int     `json:"vdd,omitempty"`
			CO2                 *int     `json:"co2,omitempty"`
			Humidity            *int     `json:"humidity,omitempty"`
			Light               *int     `json:"lights,omitempty"`
			Motion              *int     `json:"motion,omitempty"`
		} `json:"object"`
	}{}

	err := json.Unmarshal(msg, &d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal elsys payload: %s", err.Error())
	}

	pp := &Payload{
		DevEUI:     d.DevEUI,
		FPort:      strconv.Itoa(d.FPort),
		SensorType: d.SensorType,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	if d.Object.Temperature != nil {
		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			*d.Object.Temperature,
		}
		pp.Measurements = append(pp.Measurements, temp)
	}

	if d.Object.ExternalTemperature != nil {
		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			*d.Object.ExternalTemperature,
		}
		pp.Measurements = append(pp.Measurements, temp)
	}

	if d.Object.CO2 != nil {
		co2 := struct {
			CO2 int `json:"co2"`
		}{
			*d.Object.CO2,
		}
		pp.Measurements = append(pp.Measurements, co2)
	}

	if d.Object.Humidity != nil {
		hmd := struct {
			Humidity int `json:"humidity"`
		}{
			*d.Object.Humidity,
		}
		pp.Measurements = append(pp.Measurements, hmd)
	}

	if d.Object.Light != nil {
		lght := struct {
			Light int `json:"light"`
		}{
			*d.Object.Light,
		}
		pp.Measurements = append(pp.Measurements, lght)
	}

	if d.Object.Motion != nil {
		mtn := struct {
			Motion int `json:"motion"`
		}{
			*d.Object.Motion,
		}
		pp.Measurements = append(pp.Measurements, mtn)
	}

	if d.Object.Vdd != nil {
		bat := struct {
			BatteryLevel int `json:"battery_level"`
		}{
			*d.Object.Vdd, // TODO: Adjust for max VDD
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
