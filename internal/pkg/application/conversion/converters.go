package conversion

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/farshidtz/senml/v2"
)

type MessageConverterFunc func(ctx context.Context, internalID string, p payload.Payload, fn func(p senml.Pack) error) error

func PeopleCount(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	PeopleCount := func(v int) SenMLDecoratorFunc {
		return Value("1", float64(v))
	}

	if count, ok := payload.Get[int](p, "occupancy"); ok {
		return fn(NewSenMLPack(deviceID, PeopleCountURN, p.Timestamp(), PeopleCount(count)))
	} else {
		return errors.New("no people count value in payload")
	}
}

func Temperature(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	SensorValue := func(v float64) SenMLDecoratorFunc { return Value("5700", v) }

	if temp, ok := payload.Get[float64](p, "temperature"); ok {
		return fn(NewSenMLPack(deviceID, TemperatureURN, p.Timestamp(), SensorValue(temp)))
	} else {
		return errors.New("no temperature value in payload")
	}
}

func AirQuality(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	CO2 := func(v int) SenMLDecoratorFunc { return Value("17", float64(v)) }

	if c, ok := payload.Get[int](p, "co2"); ok {
		return fn(NewSenMLPack(deviceID, AirQualityURN, p.Timestamp(), CO2(c)))
	} else {
		return errors.New("no co2 value in payload")
	}
}

func Presence(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	DigitalInputState := func(vb bool) SenMLDecoratorFunc { return BoolValue("5500", vb) }

	if b, ok := payload.Get[bool](p, "presence"); ok {
		return fn(NewSenMLPack(deviceID, PresenceURN, p.Timestamp(), DigitalInputState(b)))
	} else {
		return errors.New("no presence value in payload")
	}
}

func Illuminance(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	SensorValue := func(l int) SenMLDecoratorFunc { return Value("5700", float64(l)) }

	if i, ok := payload.Get[int](p, "light"); ok {
		return fn(NewSenMLPack(deviceID, IlluminanceURN, p.Timestamp(), SensorValue(i)))
	} else {
		return errors.New("no illuminance value in payload")
	}
}

func Watermeter(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	CumulatedWaterVolume := func(v, sum float64, t time.Time) SenMLDecoratorFunc {
		return Rec("1", &v, &sum, "", &t, senml.UnitCubicMeter, nil)
	}
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

	if volumes, ok := payload.GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](p, "volume"); ok {
		for _, v := range volumes {
			volm3 := roundFloat(v.Volume * 0.001)
			summ3 := roundFloat(v.Cumulated * 0.001)
			decorators = append(decorators, CumulatedWaterVolume(volm3, summ3, v.Time))
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
		return errors.New("no water meter values in payload")
	}

	// use timestamp from sensor as default, fallback to timestamp from sensorEvent (gateway)
	var timestamp time.Time
	if ts, ok := payload.Get[time.Time](p, "timestamp"); ok {
		timestamp = ts
	} else {
		timestamp = p.Timestamp()
	}

	return fn(NewSenMLPack(deviceID, WatermeterURN, timestamp, decorators...))
}

func Pressure(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	var decorators []SenMLDecoratorFunc
	SensorValue := func(v int16) SenMLDecoratorFunc {
		f := float64(v)
		return Rec("5700", &f, nil, "", nil, "kPa", nil)
	} // TODO: kPa not in senml units

	if pressures, ok := payload.GetSlice[struct {
		Pressure int16
	}](p, "pressure"); ok {
		for _, pressure := range pressures {
			decorators = append(decorators, SensorValue(pressure.Pressure))
		}
	}

	if len(decorators) == 0 {
		return errors.New("no pressure value in payload")
	}

	return fn(NewSenMLPack(deviceID, PressureURN, p.Timestamp(), decorators...))
}

func Conductivity(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	var decorators []SenMLDecoratorFunc
	SensorValue := func(v float64) SenMLDecoratorFunc {
		return Rec("5700", &v, nil, "", nil, senml.UnitSiemensPerMeter, nil)
	}

	if resistances, ok := payload.GetSlice[struct {
		Resistance int32
	}](p, "resistance"); ok {
		for _, r := range resistances {
			if r.Resistance != 0 {
				decorators = append(decorators, SensorValue(1/float64(r.Resistance)))
			}
		}
	}

	if len(decorators) == 0 {
		return errors.New("no conductivity value in payload")
	}

	return fn(NewSenMLPack(deviceID, ConductivityURN, p.Timestamp(), decorators...))
}

func Humidity(ctx context.Context, deviceID string, p payload.Payload, fn func(p senml.Pack) error) error {
	SensorValue := func(v int) SenMLDecoratorFunc {
		f := float64(v)
		return Rec("5700", &f, nil, "", nil, senml.UnitRelativeHumidity, nil)
	}

	if h, ok := payload.Get[int](p, "humidity"); ok {
		return fn(NewSenMLPack(deviceID, HumidityURN, p.Timestamp(), SensorValue(h)))
	} else {
		return errors.New("no humidity value in payload")
	}
}
