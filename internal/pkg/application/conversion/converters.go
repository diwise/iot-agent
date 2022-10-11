package conversion

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	lwm2m "github.com/diwise/iot-core/pkg/lwm2m"
	measurements "github.com/diwise/iot-core/pkg/measurements"
	"github.com/farshidtz/senml/v2"
)

type MessageConverterFunc func(ctx context.Context, internalID string, payload decoder.Payload) (senml.Pack, error)

func Temperature(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		Measurements []struct {
			Temp *float64 `json:"temperature"`
		} `json:"measurements"`
	}{}

	if err := payload.ConvertToStruct(&dm); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %s", err.Error())
	}

	baseTime, err := parseTime(dm.Timestamp)
	if err != nil {
		return nil, err
	}

	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Temperature,
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	})

	for _, m := range dm.Measurements {
		if m.Temp != nil {
			rec := senml.Record{
				Name:  measurements.Temperature,
				Value: m.Temp,
			}

			pack = append(pack, rec)
		}
	}

	return pack, nil
}

func AirQuality(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		Measurements []struct {
			CO2 *int `json:"co2"`
		} `json:"measurements"`
	}{}

	if err := payload.ConvertToStruct(&dm); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %s", err.Error())
	}

	baseTime, err := parseTime(dm.Timestamp)
	if err != nil {
		return nil, err
	}

	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.AirQuality,
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	})

	for _, m := range dm.Measurements {
		if m.CO2 != nil {
			co2 := float64(*m.CO2)
			rec := senml.Record{
				Name:  measurements.CO2,
				Value: &co2,
			}

			pack = append(pack, rec)
		}
	}

	return pack, nil
}

func Presence(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		Measurements []struct {
			Presence *bool `json:"present"`
		} `json:"measurements"`
	}{}

	if err := payload.ConvertToStruct(&dm); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %s", err.Error())
	}

	baseTime, err := parseTime(dm.Timestamp)
	if err != nil {
		return nil, err
	}

	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Presence,
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	})

	for _, m := range dm.Measurements {
		if m.Presence != nil {
			rec := senml.Record{
				Name:      measurements.Presence,
				BoolValue: m.Presence,
			}

			pack = append(pack, rec)
		}
	}

	return pack, nil
}

func Watermeter(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		DeviceName   string `json:"deviceName"`
		Measurements []struct {
			CurrentVolume   *float64 `json:"currentVolume,omitempty"`
			CurrentDateTime *string  `json:"currentTime,omitempty"`
		} `json:"measurements"`
	}{}

	if err := payload.ConvertToStruct(&dm); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %s", err.Error())
	}

	baseTime, err := parseTime(dm.Timestamp)
	if err != nil {
		return nil, err
	}

	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Watermeter,
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	}, senml.Record{
		Name:        "DeviceName",
		StringValue: dm.DeviceName,
	})

	for _, m := range dm.Measurements {
		if m.CurrentVolume != nil {
			rec := senml.Record{
				Name:  measurements.CumulatedWaterVolume,
				Value: m.CurrentVolume,
			}

			pack = append(pack, rec)
		}

		if m.CurrentDateTime != nil {
			rec := senml.Record{
				Name:        "CurrentDateTime",
				StringValue: *m.CurrentDateTime,
			}

			pack = append(pack, rec)
		}
	}

	return pack, nil
}

func Pressure(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		Measurements []struct {
			Pressures *[]int16 `json:"pressure,omitempty"`
		} `json:"measurements"`
	}{}

	if err := payload.ConvertToStruct(&dm); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %s", err.Error())
	}

	baseTime, err := parseTime(dm.Timestamp)
	if err != nil {
		return nil, err
	}

	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Pressure,
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	})

	for _, m := range dm.Measurements {
		if m.Pressures != nil && len(*m.Pressures) > 0 {
			for _, p := range *m.Pressures {
				var pressureFloat64 = float64(p)

				rec := senml.Record{
					Name:  lwm2m.Pressure,
					Value: &pressureFloat64,
				}

				pack = append(pack, rec)
			}
		}
	}

	return pack, nil
}

func Conductivity(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		Measurements []struct {
			Conductivities *[]int32 `json:"conductivity,omitempty"`
		} `json:"measurements"`
	}{}

	if err := payload.ConvertToStruct(&dm); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %s", err.Error())
	}

	baseTime, err := parseTime(dm.Timestamp)
	if err != nil {
		return nil, err
	}

	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Conductivity,
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	})

	for _, m := range dm.Measurements {
		if m.Conductivities != nil && len(*m.Conductivities) > 0 {
			for _, c := range *m.Conductivities {
				var conductivityFloat64 = float64(c)

				rec := senml.Record{
					Name:  lwm2m.Conductivity,
					Value: &conductivityFloat64,
				}

				pack = append(pack, rec)
			}
		}
	}

	return pack, nil
}

func parseTime(t string) (float64, error) {
	baseTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return 0, fmt.Errorf("unable to parse time %s as RFC3339, %s", t, err.Error())
	}

	return float64(baseTime.Unix()), nil
}
