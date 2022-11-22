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
	/*
		ObjectURN: urn:oma:lwm2m:ext:3303
		ID      Name            Type     Unit
		5700    Sensor Value    Float
	*/

	if temp, ok := payload.Get[float64](p, "temperature"); ok {
		pack := NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3303", p.Timestamp(), Value("5700", temp))
		return fn(pack)
	} else {
		return fmt.Errorf("could not get temperature for device %s", deviceID)
	}
}

func AirQuality(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	/*
		ObjectURN: urn:oma:lwm2m:ext:3428
		ID  Name    Type    Unit
		17  CO2     Float   ppm
	*/

	if c, ok := payload.Get[int](p, "co2"); ok {
		co2 := float64(c)
		pack := NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3428", p.Timestamp(), Value("17", co2))
		return fn(pack)
	} else {
		return fmt.Errorf("could not get co2 for device %s", deviceID)
	}
}

func Presence(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	/*
		ObjectURN: urn:oma:lwm2m:ext:3302
		ID      Name                    Type       Unit
		5500    Digital Input State     Boolean
	*/

	if b, ok := payload.Get[bool](p, "presence"); ok {
		pack := NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3302", p.Timestamp(), BoolValue("5500", b))
		return fn(pack)
	} else {
		return fmt.Errorf("could not get presence for device %s", deviceID)
	}
}

func Watermeter(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	/*
		ObjectURN: urn:oma:lwm2m:ext:3424
		ID   Name                       Type        Unit
		1    Cumulated water volume     Float       m3
		3    Type of meter              String
		10   Leak detected              Boolean
		11   Back flow detected         Boolean
	*/

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
					decorators = append(decorators, Rec("1", &volm3, &summ3, "", &v.Time, senml.UnitCubicMeter, nil))
				}
			}
		}
	}

	if t, ok := payload.Get[string](p, "type"); ok {
		decorators = append(decorators, Rec("3", nil, nil, t, nil, "", nil))
	}

	t := true
	if contains(p.Status().Messages, "Leak") {
		decorators = append(decorators, Rec("10", nil, nil, "", nil, "", &t))
	}
	if contains(p.Status().Messages, "Backflow") {
		decorators = append(decorators, Rec("11", nil, nil, "", nil, "", &t))
	}

	if len(decorators) == 0 {
		return fmt.Errorf("could not get any watermeter values for device %s", deviceID)
	}

	pack := NewSenMLPack(deviceID, "urn:oma:lwm2m:ext:3424", p.Timestamp(), decorators...)
	
	return fn(pack)
}
