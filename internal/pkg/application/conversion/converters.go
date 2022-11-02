package conversion

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/measurements"
	"github.com/farshidtz/senml/v2"
)

type MessageConverterFunc func(ctx context.Context, internalID string, p payload.Payload) (senml.Pack, error)

func Temperature(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	if temp, ok := payload.Get[float64](p, measurements.Temperature); ok {
		return NewSenMLPack(deviceID, lwm2m.Temperature, p.Timestamp(), Value(measurements.Temperature, temp)), nil
	} else {
		return nil, fmt.Errorf("could not get temperature for device %s", deviceID)
	}
}

func AirQuality(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	if c, ok := payload.Get[int](p, "co2"); ok {
		co2 := float64(c)
		return NewSenMLPack(deviceID, lwm2m.AirQuality, p.Timestamp(), Value(measurements.CO2, co2)), nil
	} else {
		return nil, fmt.Errorf("could not get co2 for device %s", deviceID)
	}
}

func Presence(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	if b, ok := payload.Get[bool](p, measurements.Presence); ok {
		return NewSenMLPack(deviceID, lwm2m.Presence, p.Timestamp(), BoolValue(measurements.Presence, b)), nil
	} else {
		return nil, fmt.Errorf("could not get presence for device %s", deviceID)
	}
}

func Watermeter(ctx context.Context, deviceID string, p payload.Payload) (senml.Pack, error) {
	var decorators []SenMLDecoratorFunc

	if cv, ok := payload.Get[float64](p, "currentVolume"); ok {
		decorators = append(decorators, Value(measurements.CumulatedWaterVolume, cv))
	}

	if ct, ok := payload.Get[time.Time](p, "currentTime"); ok {
		decorators = append(decorators, Time("CurrentDateTime", ct))
	}

	if dv, ok := p.Get("deltaVolume"); ok {
		if deltas, ok := dv.([]interface{}); ok {
			for _, delta := range deltas {
				if d, ok := delta.(struct {
					Delta        float64
					Cumulated    float64
					LogValueDate time.Time
				}); ok {
					decorators = append(decorators, DeltaVolume(d.Delta, d.Cumulated, d.LogValueDate))
				}
			}
		}
	}

	if len(decorators) == 0 {
		return nil, fmt.Errorf("could not get any watermeter values for device %s", deviceID)
	}

	return NewSenMLPack(deviceID, lwm2m.Watermeter, p.Timestamp(), decorators...), nil
}
