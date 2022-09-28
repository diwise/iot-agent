package conversion

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	lwm2m "github.com/diwise/iot-core/pkg/lwm2m"
	measurements "github.com/diwise/iot-core/pkg/measurements"
	"github.com/farshidtz/senml/v2"
	//w3org "github.com/mats-dahlberg-goteborg/iot-core/pkg/glossary/w3org"
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
		DeviceName   string `json:"deviceName"`
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
		BaseName:    "urn:oma:lwm2m:ext:3323", //  	<todo> impl senare från core
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	}, senml.Record{
		Name:        "DeviceName",
		StringValue: dm.DeviceName,
	})

	for _, m := range dm.Measurements {
		if m.Pressures != nil && len(*m.Pressures) > 0 {
			for _, p := range *m.Pressures {
				var pressureFloat64 = float64(p)

				rec := senml.Record{
					Name:  "Pressure", // pkg/measurements/constants.go <todo> impl senare från core
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
		DeviceName   string `json:"deviceName"`
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
		BaseName:    "urn:oma:lwm2m:ext:3327", //  	<todo> impl senare från core
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	}, senml.Record{
		Name:        "DeviceName",
		StringValue: dm.DeviceName,
	})

	for _, m := range dm.Measurements {
		if m.Conductivities != nil && len(*m.Conductivities) > 0 {
			for _, c := range *m.Conductivities {
				var conductivityFloat64 = float64(c)

				rec := senml.Record{
					Name:  "Conductivity", // pkg/measurements/constants.go <todo> impl senare från core
					Value: &conductivityFloat64,
				}

				pack = append(pack, rec)
			}
		}
	}

	return pack, nil
}

/*
func SoilMoisture(ctx context.Context, deviceID string, payload decoder.Payload) (senml.Pack, error) {
	dm := struct {
		Timestamp    string `json:"timestamp"`
		DeviceName   string `json:"deviceName"`
		Measurements []struct {
			TransmissionReason *int8    `json:"transmissionReason,omitempty"`
			ProtocolVersion    *int16   `json:"protocolVersion,omitempty"`
			BatteryVoltage     *float64 `json:"battery,omitempty"`
			Resistances        *[]int32 `json:"conductivity,omitempty"`
			SoilMoistures      *[]int16 `json:"preassure,omitempty"`
			Temperature        *float64 `json:"temperature"`
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
		BaseName:    "SoilMoisture", //  	<todo> impl senare från core
		BaseTime:    baseTime,
		Name:        "0",
		StringValue: deviceID,
	}, senml.Record{
		Name:        "DeviceName",
		StringValue: dm.DeviceName,
	})

	for _, m := range dm.Measurements {
		if m.TransmissionReason != nil {
			var transmissionReasonInt8 = int8(*m.TransmissionReason)
			var transmissionReasonFloat64 = float64(transmissionReasonInt8)

			rec := senml.Record{
				Name:  "Transmisison reason", // pkg/measurements/constants.go <todo> impl senare från core
				Value: &transmissionReasonFloat64,
			}

			pack = append(pack, rec)
		}

		if m.ProtocolVersion != nil {
			var protocolVersionInt16 = int16(*m.ProtocolVersion)
			var protocolVersionFloat64 = float64(protocolVersionInt16)

			rec := senml.Record{
				Name:  "Protocol version", // pkg/measurements/constants.go <todo> impl senare från core
				Value: &protocolVersionFloat64,
			}

			pack = append(pack, rec)
		}

		if m.BatteryVoltage != nil {
			rec := senml.Record{
				Name:  measurements.BatteryLevel, // pkg/measurements/constants.go <todo> impl senare från core
				Value: m.BatteryVoltage,
			}

			pack = append(pack, rec)
		}

		if m.Resistances != nil && len(*m.Resistances) > 0 {
			for _, r := range *m.Resistances {
				var resistanceFloat64 = float64(r)

				rec := senml.Record{
					Name:  "Conductivity", // pkg/measurements/constants.go <todo> impl senare från core
					Value: &resistanceFloat64,
				}

				pack = append(pack, rec)
			}
		}

		if m.SoilMoistures != nil && len(*m.SoilMoistures) > 0 {
			for _, r := range *m.SoilMoistures {
				var soilMoistureFloat64 = float64(r)

				rec := senml.Record{
					Name:  "Preassure", // pkg/measurements/constants.go <todo> impl senare från core
					Value: &soilMoistureFloat64,
				}

				pack = append(pack, rec)
			}
		}

		if m.Temperature != nil {
			rec := senml.Record{
				Name:  measurements.Temperature,
				Value: m.Temperature,
			}

			pack = append(pack, rec)
		}
	}

	return pack, nil
}
*/

func parseTime(t string) (float64, error) {
	baseTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return 0, fmt.Errorf("unable to parse time %s as RFC3339, %s", t, err.Error())
	}

	return float64(baseTime.Unix()), nil
}
