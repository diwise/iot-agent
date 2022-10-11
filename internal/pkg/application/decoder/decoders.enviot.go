package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func EnviotDecoder(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {

	d := struct {
		DevEUI     string `json:"devEUI"`
		FCnt       int    `json:"fCnt"`
		FPort      int    `json:"fPort"`
		SensorType string `json:"deviceProfileName"`
		Data       string `json:"data"`
		Object     struct {
			Payload struct {
				Battery      *int     `json:"battery,omitempty"`
				Humidity     *int     `json:"humidity,omitempty"`
				SensorStatus int      `json:"sensorStatus"`
				SnowHeight   *int     `json:"snowHeight,omitempty"`
				Temperature  *float32 `json:"temperature,omitempty"`
			} `json:"payload"`
		} `json:"object"`
	}{}

	err := json.Unmarshal(msg, &d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal enviot payload: %s", err.Error())
	}

	pp := &Payload{
		DevEUI:     d.DevEUI,
		FPort:      strconv.Itoa(d.FPort),
		SensorType: d.SensorType,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	if d.Object.Payload.Temperature != nil {
		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			*d.Object.Payload.Temperature,
		}
		pp.Measurements = append(pp.Measurements, temp)
	}

	if d.Object.Payload.Battery != nil {
		bat := struct {
			BatteryLevel int `json:"battery_level"`
		}{
			*d.Object.Payload.Battery,
		}
		pp.BatteryLevel = bat.BatteryLevel
		pp.Measurements = append(pp.Measurements, bat)
	}

	if d.Object.Payload.Humidity != nil {
		hmd := struct {
			Humidity int `json:"humidity"`
		}{
			*d.Object.Payload.Humidity,
		}
		pp.Measurements = append(pp.Measurements, hmd)
	}

	if d.Object.Payload.SensorStatus == 0 && d.Object.Payload.SnowHeight != nil {
		snow := struct {
			SnowHeight int `json:"snow_height"`
		}{
			*d.Object.Payload.SnowHeight,
		}
		pp.Measurements = append(pp.Measurements, snow)
	}

	return fn(ctx, *pp)
}
