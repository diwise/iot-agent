package conversion

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-core/pkg/lwm2m"
	"github.com/diwise/iot-core/pkg/measurements"
	"github.com/farshidtz/senml/v2"
)

type MessageConverterFunc func(ctx context.Context, internalID string, p payload.Payload, fn func(p senml.Pack) error) error

func Temperature(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	if temp, ok := payload.Get[float64](p, measurements.Temperature); ok {
		pack := NewSenMLPack(deviceID, lwm2m.Temperature, p.Timestamp(), Value(measurements.Temperature, temp))
		return fn(pack)
	} else {
		return fmt.Errorf("could not get temperature for device %s", deviceID)
	}
}

func AirQuality(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	if c, ok := payload.Get[int](p, "co2"); ok {
		co2 := float64(c)
		pack := NewSenMLPack(deviceID, lwm2m.AirQuality, p.Timestamp(), Value(measurements.CO2, co2))
		return fn(pack)
	} else {
		return fmt.Errorf("could not get co2 for device %s", deviceID)
	}
}

func Presence(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	if b, ok := payload.Get[bool](p, measurements.Presence); ok {
		pack := NewSenMLPack(deviceID, lwm2m.Presence, p.Timestamp(), BoolValue(measurements.Presence, b))
		return fn(pack)
	} else {
		return fmt.Errorf("could not get presence for device %s", deviceID)
	}
}

func Watermeter(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	var decorators []SenMLDecoratorFunc

	roundFloat := func(val float64, precision uint) float64 {
		ratio := math.Pow(10, float64(precision))
		return math.Round(val*ratio) / ratio
	}

	if cv, ok := payload.Get[float64](p, "currentVolume"); ok {
		volm3 := roundFloat(cv*0.001, 3)
		decorators = append(decorators, Value("CurrentVolume", volm3))
	}

	if lv, ok := payload.Get[float64](p, "logVolume"); ok {
		volm3 := roundFloat(lv*0.001, 3)
		decorators = append(decorators, Value("LogVolume", volm3))
	}

	if ct, ok := payload.Get[time.Time](p, "currentTime"); ok {
		decorators = append(decorators, Time("CurrentDateTime", ct))
	}

	if lt, ok := payload.Get[time.Time](p, "logDateTime"); ok {
		decorators = append(decorators, Time("LogDateTime", lt))
	}

	if t, ok := payload.Get[float64](p, "temperature"); ok {
		decorators = append(decorators, Value(measurements.Temperature, t*0.01))
	}

	if dv, ok := p.Get("deltaVolume"); ok {
		if deltas, ok := dv.([]interface{}); ok {
			for _, delta := range deltas {
				if d, ok := delta.(struct {
					Delta        float64
					Cumulated    float64
					LogValueDate time.Time
				}); ok {
					decorators = append(decorators, DeltaVolume(d.Delta*0.001, d.Cumulated*0.001, d.LogValueDate))
				}
			}
		}
	}

	if len(decorators) == 0 {
		return fmt.Errorf("could not get any watermeter values for device %s", deviceID)
	}

	pack := NewSenMLPack(deviceID, lwm2m.Watermeter, p.Timestamp(), decorators...)
	return fn(pack)
}
