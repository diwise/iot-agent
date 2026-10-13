package mapper

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/diwise/iot-agent/internal/application/types"
)

// TransformFunc converts one value. It accepts and returns only
// JSON-domain values.
type TransformFunc func(v any) (any, error)

var (
	ErrUnknownTransform       = errors.New("unknown transform")
	ErrInvalidTransformParams = errors.New("invalid transform params")
)

// GetTransform builds a validated transform. Unknown names and bad
// params fail here, never at mapping time.
func GetTransform(name string, params map[string]any) (TransformFunc, error) {
	switch name {
	case "identity":
		if len(params) > 0 {
			return nil, fmt.Errorf("%w: identity takes no params", ErrInvalidTransformParams)
		}
		return func(v any) (any, error) { return v, nil }, nil
	case "scale":
		factor, divisor, offset, err := scaleParams(params)
		if err != nil {
			return nil, err
		}
		return func(v any) (any, error) {
			f, err := types.AsFloat(v)
			if err != nil {
				return nil, err
			}
			return f*factor/divisor + offset, nil
		}, nil
	case "reciprocal":
		if len(params) > 0 {
			return nil, fmt.Errorf("%w: reciprocal takes no params", ErrInvalidTransformParams)
		}
		return func(v any) (any, error) {
			f, err := types.AsFloat(v)
			if err != nil {
				return nil, err
			}
			if f == 0 {
				return nil, fmt.Errorf("reciprocal: division by zero")
			}
			return 1 / f, nil
		}, nil
	case "to_int":
		if len(params) > 0 {
			return nil, fmt.Errorf("%w: to_int takes no params", ErrInvalidTransformParams)
		}
		return func(v any) (any, error) {
			f, err := types.AsFloat(v)
			if err != nil {
				return nil, err
			}
			if f > float64(maxIntFloat()) || f < float64(minIntFloat()) {
				return nil, fmt.Errorf("to_int: %v overflows int", f)
			}
			return float64(int(f)), nil
		}, nil
	case "equals":
		want, ok := params["value"]
		if !ok || len(params) != 1 {
			return nil, fmt.Errorf("%w: equals needs exactly one value param", ErrInvalidTransformParams)
		}
		if want == nil {
			return nil, fmt.Errorf("%w: equals value must not be null", ErrInvalidTransformParams)
		}
		return func(v any) (any, error) {
			if v == nil {
				return nil, fmt.Errorf("equals: null input")
			}
			if wf, err := types.AsFloat(want); err == nil {
				vf, err := types.AsFloat(v)
				if err != nil {
					return nil, err
				}
				return vf == wf, nil
			}
			return reflect.DeepEqual(v, want), nil
		}, nil
	case "magnet_open_to_bool":
		if len(params) > 0 {
			return nil, fmt.Errorf("%w: magnet_open_to_bool takes no params", ErrInvalidTransformParams)
		}
		return func(v any) (any, error) {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("magnet_open_to_bool: expected string, got %T", v)
			}
			switch {
			case strings.EqualFold(s, "open"):
				return true, nil
			case strings.EqualFold(s, "close"):
				return false, nil
			default:
				return nil, fmt.Errorf("magnet_open_to_bool: unknown state %q", s)
			}
		}, nil
	case "vdd_to_soc":
		if len(params) > 0 {
			return nil, fmt.Errorf("%w: vdd_to_soc takes no params", ErrInvalidTransformParams)
		}
		return func(v any) (any, error) {
			mv, err := types.AsInt(v)
			if err != nil {
				return nil, err
			}
			return VDDToSOC(mv), nil
		}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownTransform, name)
	}
}

func scaleParams(params map[string]any) (factor, divisor, offset float64, err error) {
	factor, divisor, offset = 1, 1, 0
	for key, want := range map[string]*float64{"factor": &factor, "divisor": &divisor, "offset": &offset} {
		raw, ok := params[key]
		if !ok {
			continue
		}
		f, ferr := types.AsFloat(raw)
		if ferr != nil {
			return 0, 0, 0, fmt.Errorf("%w: scale %s is not numeric", ErrInvalidTransformParams, key)
		}
		*want = f
	}
	for key := range params {
		if key != "factor" && key != "divisor" && key != "offset" {
			return 0, 0, 0, fmt.Errorf("%w: scale has unknown param %q", ErrInvalidTransformParams, key)
		}
	}
	if divisor == 0 {
		return 0, 0, 0, fmt.Errorf("%w: scale divisor must not be zero", ErrInvalidTransformParams)
	}
	return factor, divisor, offset, nil
}

func maxIntFloat() int {
	return int(^uint(0) >> 1)
}

func minIntFloat() int {
	return -maxIntFloat() - 1
}

// VDDToSOC estimates battery percent from millivolts using the shared
// Elsys curve. Elsys and X2Climate converters delegate to this
// implementation after migration. The result is an estimate, not a
// measurement.
func VDDToSOC(vdd int) int {
	halfVoltageCapacity := 3.4177
	calibrationFactor := 21.347

	batteryVolt := math.Round(float64(vdd)/10) / 100

	if batteryVolt >= 3.6 {
		return 100
	}

	if batteryVolt < 3.3 {
		return 0
	}

	soc := 100 * (1 / (1 + math.Exp(-calibrationFactor*(batteryVolt-halfVoltageCapacity))))
	roundedSoc := math.Round(soc*100) / 100
	return int(roundedSoc)
}
