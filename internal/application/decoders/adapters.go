package decoders

import (
	"context"
	"math"
	"strings"

	"github.com/diwise/iot-agent/internal/application/decoders/axsensor"
	"github.com/diwise/iot-agent/internal/application/decoders/elsys"
	"github.com/diwise/iot-agent/internal/application/decoders/enviot"
	"github.com/diwise/iot-agent/internal/application/decoders/mapper"
	"github.com/diwise/iot-agent/internal/application/decoders/niab"
	"github.com/diwise/iot-agent/internal/application/decoders/sensative"
	"github.com/diwise/iot-agent/internal/application/decoders/sensefarm"
	"github.com/diwise/iot-agent/internal/application/decoders/vegapuls"
	"github.com/diwise/iot-agent/internal/application/decoders/x2climate"
	"github.com/diwise/iot-agent/internal/application/types"
)

// adapterForGoRef selects the per-sensor adapter that owns legacy
// gating, errors and normalization. Unknown refs use generic marshal.
func adapterForGoRef(goRef string, fn DecoderFunc) GenericDecoderFunc {
	switch strings.ToLower(goRef) {
	case "elsys", "elt_2_hp", "elsys_codec", "elsys/elt/sht3x":
		return AdaptElsys(fn)
	case "enviot":
		return AdaptEnviot(fn)
	case "niab-fls":
		return AdaptNiab(fn)
	case "vegapuls_air_41":
		return AdaptVegapuls(fn)
	case "axsensor":
		return AdaptAxsensor(fn)
	case "x2climate":
		return AdaptX2Climate(fn)
	case "sensefarm", "cube02":
		return AdaptSensefarm(fn)
	case "sensative", "strips_lora_ms_h", "presence":
		return AdaptSensative(fn)
	default:
		return AdaptLegacy(fn)
	}
}

// copyMapping returns an isolated copy for mapping adaptations.
// JSON-domain values (maps, lists, scalars) are deep-copied so
// decoderdata for storage never changes through the mapping map.
func copyMapping(data types.GenericPayload) types.GenericPayload {
	out := make(types.GenericPayload, len(data))
	for k, v := range data {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			m[k] = deepCopyValue(e)
		}
		return m
	case []any:
		l := make([]any, len(t))
		for i, e := range t {
			l[i] = deepCopyValue(e)
		}
		return l
	default:
		return v
	}
}

// AdaptElsys restores float32 fields with direct conversion so the
// mapper sees exactly the legacy converter input. JSON roundtrip
// alone can shift float32 values before LWM2M rounding.
func AdaptElsys(fn DecoderFunc) GenericDecoderFunc {
	base := AdaptLegacy(fn)
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		typed, ok := p.(elsys.ElsysPayload)
		if !ok {
			return base(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		if typed.Temperature != nil {
			mapping["temperature"] = float64(*typed.Temperature)
		}
		if typed.ExternalTemperature != nil {
			mapping["externalTemperature"] = float64(*typed.ExternalTemperature)
		}
		if typed.ExternalTemperature2 != nil {
			mapping["externalTemperature2"] = float64(*typed.ExternalTemperature2)
		}
		if typed.Lat != nil {
			mapping["lat"] = float64(*typed.Lat)
		}
		if typed.Lon != nil {
			mapping["long"] = float64(*typed.Lon)
		}
		if typed.Pressure != nil {
			mapping["pressure"] = float64(*typed.Pressure)
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  p.BatteryLevel(),
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// AdaptEnviot keeps raw snowHeight for storage but omits it from
// mapping when sensorStatus != 0, matching legacy converter gating.
func AdaptEnviot(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		typed, ok := p.(enviot.EnviotPayload)
		if !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		if typed.Payload.SensorStatus != 0 {
			if nested, ok := mapping["payload"].(map[string]any); ok {
				delete(nested, "snowHeight")
			}
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  p.BatteryLevel(),
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// AdaptNiab restores valid zero battery values that JSON omits.
// The decoder already returns ErrSensorReadingError for distance -1.
func AdaptNiab(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		typed, ok := p.(niab.NiabPayload)
		if !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		// omitzero drops battery 0; mapping needs the valid zero.
		if _, ok := mapping["battery"]; !ok {
			mapping["battery"] = float64(typed.Battery)
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  p.BatteryLevel(),
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// AdaptVegapuls normalizes meter and Celsius in a separate mapping
// map. Raw data and unit fields stay in storage unchanged.
func AdaptVegapuls(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		typed, ok := p.(vegapuls.VegapulsPayload)
		if !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		if d, ok := vegapulsNormalizedDistance(typed); ok {
			mapping["distance_m"] = roundFloat5(d)
		}
		if t, ok := vegapulsNormalizedTemperature(typed); ok {
			mapping["temperature_c"] = t
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  p.BatteryLevel(),
			Code:     code,
			Messages: messages,
		}, nil
	}
}

func vegapulsNormalizedDistance(p vegapuls.VegapulsPayload) (float64, bool) {
	if p.Distance == nil {
		return 0, false
	}
	d := *p.Distance
	if p.Unit != nil {
		switch *p.Unit {
		case 44:
			d *= 0.3048
		case 47:
			d *= 0.0254
		case 49:
			d /= 1000
		case 45:
		}
	}
	if math.IsNaN(d) || math.IsInf(d, 0) {
		return 0, false
	}
	return d, true
}

func vegapulsNormalizedTemperature(p vegapuls.VegapulsPayload) (float64, bool) {
	if p.Temperature == nil {
		return 0, false
	}
	t := *p.Temperature
	if p.TemperatureUnit != nil && *p.TemperatureUnit == 33 {
		t = (t - 32) * 5 / 9
	}
	if math.IsNaN(t) || math.IsInf(t, 0) {
		return 0, false
	}
	return t, true
}

func roundFloat5(v float64) float64 {
	r := math.Pow(10, 5)
	return math.Round(v*r) / r
}

// AdaptAxsensor adds exact legacy level math as mapping keys.
// Namngivna transforms kan inte uttrycka avrundningen exakt.
// Vbat är spänning (mV), inte procent: status batteri stannar nil,
// spänning mappas som spänning (Device/7).
func AdaptAxsensor(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		typed, ok := p.(axsensor.AxsensorPayload)
		if !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		if typed.Level != nil {
			perc := roundFloat5(*typed.Level * 100 / 1400)
			mapping["level_percent"] = perc
			mapping["level_cm"] = float64(int64((*typed.Level + 5) / 10))
		}
		if typed.Distance != nil {
			mapping["distance_m"] = *typed.Distance / 1000
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  nil,
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// AdaptX2Climate applies the decided status fix: SOC from the shared
// curve only when valid voltage exists. Zero voltage gives nil
// battery percent. Voltage stays as voltage for Device/7.
func AdaptX2Climate(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		typed, ok := p.(x2climate.X2ClimatePayload)
		if !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		var battery *int
		if typed.VoltageMV != 0 {
			soc := mapper.VDDToSOC(int(typed.VoltageMV))
			battery = &soc
			mapping["battery_level"] = float64(soc)
		} else {
			delete(mapping, "battery_level")
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  battery,
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// AdaptSensefarm keeps raw lists and voltage. No SOC curve is
// selected in FAS1, so battery percent stays nil. Pressure
// overflow is fixed in mapping via float scaling, not int16 math.
func AdaptSensefarm(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		if _, ok := p.(sensefarm.SensefarmPayload); !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		mapping := copyMapping(data)
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  mapping,
			Battery:  nil,
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// AdaptSensative preserves the check-in flag for header_only mapping.
func AdaptSensative(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		if _, ok := p.(sensative.SensativePayload); !ok {
			return AdaptLegacy(fn)(ctx, e)
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  data,
			Battery:  p.BatteryLevel(),
			Code:     code,
			Messages: messages,
		}, nil
	}
}
