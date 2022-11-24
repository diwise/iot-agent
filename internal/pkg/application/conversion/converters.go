package conversion

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/farshidtz/senml/v2"
)

type MessageConverterFunc func(ctx context.Context, internalID string, p payload.Payload, fn func(p senml.Pack) error) error

func Temperature(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {	
	SensorValue := func(v float64) SenMLDecoratorFunc { return Value("5700", v) }

	if temp, ok := payload.Get[float64](p, "temperature"); ok {
		return fn(NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3303", p.Timestamp(), SensorValue(temp)))
	} else {
		return fmt.Errorf("could not get temperature for device %s", deviceID)
	}
}

func AirQuality(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	CO2 := func(v int) SenMLDecoratorFunc { return Value("17", float64(v)) }

	if c, ok := payload.Get[int](p, "co2"); ok {
		return fn(NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3428", p.Timestamp(), CO2(c)))
	} else {
		return fmt.Errorf("could not get co2 for device %s", deviceID)
	}
}

func Presence(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {	
	DigitalInputState := func(vb bool) SenMLDecoratorFunc { return BoolValue("5500", vb) }

	if b, ok := payload.Get[bool](p, "presence"); ok {
		return fn(NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3302", p.Timestamp(), DigitalInputState(b)))
	} else {
		return fmt.Errorf("could not get presence for device %s", deviceID)
	}
}

func Watermeter(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {	
	CumulatedWaterVolume := func(v, sum float64, t time.Time) SenMLDecoratorFunc { return Rec("1", &v, &sum, "", &t, senml.UnitCubicMeter, nil)}
	TypeOfMeter := func(vs string) SenMLDecoratorFunc { return Rec("3", nil, nil, vs, nil, "", nil) }
	LeakDetected := func(vb bool) SenMLDecoratorFunc { return BoolValue("10", vb) }
	BackFlowDetected := func(vb bool) SenMLDecoratorFunc { return BoolValue("11", vb) }

	var decorators []SenMLDecoratorFunc

	roundFloat := func(val float64) float64 {
		ratio := math.Pow(10, float64(3))
		return math.Round(val*ratio) / ratio
	}

	contains := func(s []string, str string) bool {
		for _, v := range s {
			if strings.EqualFold(v, str) {
				return true
			}
		}
		return false
	}

	if volumeMeasurements, ok := p.Get("volume"); ok {
		if volumes, ok := volumeMeasurements.([]interface{}); ok {
			for _, vol := range volumes {
				if v, ok := vol.(struct {
					Volume    float64
					Cumulated float64
					Time      time.Time
				}); ok {
					volm3 := roundFloat(v.Volume * 0.001)
					summ3 := roundFloat(v.Cumulated * 0.001)
					decorators = append(decorators, CumulatedWaterVolume(volm3, summ3, v.Time))
				}
			}
		}
	}

	if t, ok := payload.Get[string](p, "type"); ok {
		decorators = append(decorators, TypeOfMeter(t))
	}
	
	if contains(p.Status().Messages, "Leak") {
		decorators = append(decorators, LeakDetected(true))
	}
	if contains(p.Status().Messages, "Backflow") {
		decorators = append(decorators, BackFlowDetected(true))
	}

	if len(decorators) == 0 {
		return fmt.Errorf("could not get any watermeter values for device %s", deviceID)
	}

	// use timestamp from sensor as default, fallback to timestamp from sensorEvent (gateway)
	var timestamp time.Time
	if ts, ok := payload.Get[time.Time](p, "timestamp"); ok {
		timestamp = ts
	} else {
		timestamp = p.Timestamp()
	}

	return fn(NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3424", timestamp, decorators...))
}

func Pressure(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	var decorators []SenMLDecoratorFunc

	if sm, ok := p.Get("soilMoisture"); ok {
		if pressures, ok := sm.(struct {
			SoilMoisture []int16
		}); ok {
			for _, pressure := range pressures.SoilMoisture {
				decorators = append(decorators, Value("Pressure", float64(pressure)))
			}
		}
	}

	if len(decorators) == 0 {
		return fmt.Errorf("could not get any pressure values for device %s", deviceID)
	}

	pack := NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3323", p.Timestamp(), decorators...)
	return fn(pack)
}

func Conductivity(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	var decorators []SenMLDecoratorFunc

	if r, ok := p.Get("resistance"); ok {
		if resistances, ok := r.(struct {
			Resistance []int32
		}); ok {
			for _, resistance := range resistances.Resistance {
				decorators = append(decorators, Value("Conductivity", 1/float64(resistance)))
			}
		}
	}
	if len(decorators) == 0 {
		return fmt.Errorf("could not get any conductivity values for device %s", deviceID)
	}

	pack := NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3327", p.Timestamp(), decorators...)
	return fn(pack)
}
