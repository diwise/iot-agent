package conversion

import (
	"context"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	lwm2m "github.com/diwise/iot-core/pkg/lwm2m"
	measurements "github.com/diwise/iot-core/pkg/measurements"
	"github.com/farshidtz/senml/v2"
)

type MessageConverterFunc func(ctx context.Context, internalID string, p payload.Payload) (senml.Pack, error)

func Temperature(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Temperature,
		BaseTime:    float64(p.Timestamp().Unix()),
		Name:        "0",
		StringValue: deviceID,
	})

	if temp, ok := payload.Get[float32](p, measurements.Temperature); ok {
		t := float64(temp)
		pack = append(pack, senml.Record{
			Name:  measurements.Temperature,
			Value: &t,
		})
	}

	return pack, nil
}

func AirQuality(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.AirQuality,
		BaseTime:    float64(p.Timestamp().Unix()),
		Name:        "0",
		StringValue: deviceID,
	})

	if c, ok := payload.Get[int](p, "co2"); ok {
		co2 := float64(c)
		pack = append(pack, senml.Record{
			Name:  measurements.CO2,
			Value: &co2,
		})
	}

	return pack, nil
}

func Presence(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Presence,
		BaseTime:    float64(p.Timestamp().Unix()),
		Name:        "0",
		StringValue: deviceID,
	})

	if b, ok := payload.Get[bool](p, measurements.Presence); ok {
		pack = append(pack, senml.Record{
			Name:      measurements.Presence,
			BoolValue: &b,
		})
	}

	return pack, nil
}

func Watermeter(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	var pack senml.Pack
	pack = append(pack, senml.Record{
		BaseName:    lwm2m.Watermeter,
		BaseTime:    float64(p.Timestamp().Unix()),
		Name:        "0",
		StringValue: deviceID,
	})

	if cv, ok := payload.Get[float64](p, "currentVolume"); ok {
		pack = append(pack, senml.Record{
			Name:  measurements.CumulatedWaterVolume,
			Value: &cv,
		})
	}

	if ct, ok := payload.Get[time.Time](p, "currentTime"); ok {
		pack = append(pack, senml.Record{
			Name:        "CurrentDateTime",
			StringValue: ct.Format(time.RFC3339Nano),
			Time:        float64(ct.UnixMilli() / 1000),
		})
	}

	if dv, ok := p.Get("deltaVolume"); ok {
		// TODO: ugly code begins here... :(
		if deltas, ok := dv.([]interface{}); ok {
			for _, delta := range deltas {
				if d, ok := delta.(struct {
					Delta        float64
					Cumulated    float64
					LogValueDate time.Time
				}); ok {
					pack = append(pack, senml.Record{
						Name:  "DeltaVolume",
						Value: &d.Delta,
						Time:  float64(d.LogValueDate.UnixMilli() / 1000),
						Sum:   &d.Cumulated,
					})
				}
			}

		}
	}

	return pack, nil
}
