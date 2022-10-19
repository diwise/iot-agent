package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
)

func EnviotDecoder(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	pp := Payload{
		DevEUI:     ue.DevEui,
		SensorType: ue.SensorType,
		Timestamp:  ue.Timestamp.Format(time.RFC3339Nano),
	}

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

	if obj.Payload.Temperature != nil {
		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			*obj.Payload.Temperature,
		}
		pp.Measurements = append(pp.Measurements, temp)
	}

	if obj.Payload.Battery != nil {
		bat := struct {
			BatteryLevel int `json:"battery_level"`
		}{
			*obj.Payload.Battery,
		}
		pp.BatteryLevel = bat.BatteryLevel
		pp.Measurements = append(pp.Measurements, bat)
	}

	if obj.Payload.Humidity != nil {
		hmd := struct {
			Humidity int `json:"humidity"`
		}{
			*obj.Payload.Humidity,
		}
		pp.Measurements = append(pp.Measurements, hmd)
	}

	if obj.Payload.SensorStatus == 0 && obj.Payload.SnowHeight != nil {
		snow := struct {
			SnowHeight int `json:"snow_height"`
		}{
			*obj.Payload.SnowHeight,
		}
		pp.Measurements = append(pp.Measurements, snow)
	}

	pp.SetStatus(obj.Payload.SensorStatus, nil)

	return fn(ctx, pp)
}
