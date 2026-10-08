package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// GenericPayload holds decoded fields with their names and nesting.
// Go adapters fill it by marshaling the typed payload struct.
// JS adapters fill it from the script output unchanged.
type GenericPayload map[string]any

// DecodedResult is the generic decoder boundary.
// Data holds decoderdata for storage in the payload column.
// Mapping holds the mapper input. It may share Data when no adaptation
// is needed. Adapters must copy before changing shared maps or lists.
// Battery, Code, Messages and Warnings never enter decoderdata.
type DecodedResult struct {
	Data     GenericPayload
	Mapping  GenericPayload
	Battery  *int
	Code     string
	Messages []string
	Warnings []string
}

// BatteryLevel implements SensorPayload.
func (d DecodedResult) BatteryLevel() *int {
	return d.Battery
}

// Error implements SensorPayload.
func (d DecodedResult) Error() (string, []string) {
	return d.Code, d.Messages
}

// MarshalJSON writes only decoderdata, never status or diagnostics.
func (d DecodedResult) MarshalJSON() ([]byte, error) {
	if d.Data == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]any(d.Data))
}

// UnmarshalJSON reads stored decoderdata back into Data and Mapping.
func (d *DecodedResult) UnmarshalJSON(b []byte) error {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	d.Data = m
	d.Mapping = m
	return nil
}

// Lookup errors classify path failures. Missing keys, null parents,
// type mismatches and out-of-range indexes are not errors.
var (
	ErrInvalidPath  = errors.New("invalid lookup path")
	ErrAmbiguousKey = errors.New("ambiguous lookup key")
	ErrNotNumeric   = errors.New("value is not numeric")
	ErrNotIntegral  = errors.New("value is not an integer")
	ErrNotBool      = errors.New("value is not a bool")
)

type pathSegment struct {
	field   string
	index   int
	isIndex bool
}

// parseLookupPath parses punktnotation with backslash escaping.
// A backslash escapes one of `.[]\`. `[n]` holds a decimal index.
func parseLookupPath(path string) ([]pathSegment, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	var segs []pathSegment
	var cur strings.Builder
	flush := func() error {
		if cur.Len() == 0 {
			return fmt.Errorf("%w: empty segment in %q", ErrInvalidPath, path)
		}
		segs = append(segs, pathSegment{field: cur.String()})
		cur.Reset()
		return nil
	}

	i := 0
	trailingDot := false
	for i < len(path) {
		c := path[i]
		switch c {
		case '\\':
			if i+1 >= len(path) {
				return nil, fmt.Errorf("%w: trailing backslash in %q", ErrInvalidPath, path)
			}
			next := path[i+1]
			if next != '.' && next != '\\' && next != '[' && next != ']' {
				return nil, fmt.Errorf("%w: bad escape %q in %q", ErrInvalidPath, `\`+string(next), path)
			}
			cur.WriteByte(next)
			trailingDot = false
			i += 2
		case '.':
			if cur.Len() > 0 {
				if err := flush(); err != nil {
					return nil, err
				}
			} else if len(segs) == 0 || !segs[len(segs)-1].isIndex {
				return nil, fmt.Errorf("%w: empty segment in %q", ErrInvalidPath, path)
			}
			trailingDot = true
			i++
		case '[':
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("%w: unclosed index in %q", ErrInvalidPath, path)
			}
			digits := path[i+1 : i+end]
			if digits == "" {
				return nil, fmt.Errorf("%w: empty index in %q", ErrInvalidPath, path)
			}
			n, err := strconv.Atoi(digits)
			if err != nil || n < 0 || digits != strconv.Itoa(n) {
				return nil, fmt.Errorf("%w: bad index %q in %q", ErrInvalidPath, digits, path)
			}
			if cur.Len() > 0 {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			segs = append(segs, pathSegment{index: n, isIndex: true})
			trailingDot = false
			i += end + 1
		default:
			cur.WriteByte(c)
			trailingDot = false
			i++
		}
	}
	if cur.Len() > 0 {
		if err := flush(); err != nil {
			return nil, err
		}
		return segs, nil
	}
	if trailingDot || len(segs) == 0 {
		return nil, fmt.Errorf("%w: empty segment in %q", ErrInvalidPath, path)
	}
	return segs, nil
}

// ValidatePath reports whether path parses. It performs no lookup.
func ValidatePath(path string) error {
	_, err := parseLookupPath(path)
	return err
}

// Lookup resolves a punktnotation path against the payload.
// Exact field match wins. Otherwise a single case-insensitive match
// wins per segment. Array indexes always match exactly.
// It returns found=false for missing keys, null parents, type
// mismatches and out-of-range indexes. Explicit null returns
// found=true with a nil value. Multiple case-insensitive candidates
// return an ErrAmbiguousKey error. Lookup never panics on nil maps.
func (p GenericPayload) Lookup(path string) (any, bool, error) {
	segs, err := parseLookupPath(path)
	if err != nil {
		return nil, false, err
	}

	var cur any = map[string]any(p)
	for _, s := range segs {
		if s.isIndex {
			arr, ok := cur.([]any)
			if !ok || s.index >= len(arr) {
				return nil, false, nil
			}
			cur = arr[s.index]
			continue
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false, nil
		}
		if v, ok := m[s.field]; ok {
			cur = v
			continue
		}
		var hits []string
		for k := range m {
			if strings.EqualFold(k, s.field) {
				hits = append(hits, k)
			}
		}
		switch len(hits) {
		case 0:
			return nil, false, nil
		case 1:
			cur = m[hits[0]]
		default:
			sort.Strings(hits)
			return nil, false, fmt.Errorf("%w: %q matches %q", ErrAmbiguousKey, path, hits)
		}
	}
	return cur, true, nil
}

// AsFloat accepts Go numbers and json.Number. It rejects strings,
// bools, nil, maps, slices and non-finite values.
func AsFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, fmt.Errorf("%w: non-finite float64", ErrNotNumeric)
		}
		return n, nil
	case float32:
		f := float64(n)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, fmt.Errorf("%w: non-finite float32", ErrNotNumeric)
		}
		return f, nil
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint8:
		return float64(n), nil
	case uint16:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, fmt.Errorf("%w: %q", ErrNotNumeric, n.String())
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, fmt.Errorf("%w: non-finite json.Number", ErrNotNumeric)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("%w: %T", ErrNotNumeric, v)
	}
}

// maxExactFloatInt bounds integers that float64 represents exactly.
const maxExactFloatInt = float64(int64(1) << 53)

// AsInt accepts integral numbers with exact float64 representation
// inside the platform int range. It checks json.Number integers
// before any float64 conversion. Strings, bools, fractions,
// precision loss and overflow return an error.
func AsInt(v any) (int, error) {
	if jn, ok := v.(json.Number); ok {
		if i, err := jn.Int64(); err == nil {
			if i > int64(maxPlatformInt()) || i < int64(minPlatformInt()) {
				return 0, fmt.Errorf("%w: %q overflows int", ErrNotIntegral, jn.String())
			}
			return int(i), nil
		}
		return 0, fmt.Errorf("%w: %q", ErrNotIntegral, jn.String())
	}
	f, err := AsFloat(v)
	if err != nil {
		return 0, err
	}
	if math.Trunc(f) != f {
		return 0, fmt.Errorf("%w: %v is fractional", ErrNotIntegral, f)
	}
	if f > maxExactFloatInt || f < -maxExactFloatInt {
		return 0, fmt.Errorf("%w: %v loses precision", ErrNotIntegral, f)
	}
	if f > float64(maxPlatformInt()) || f < float64(minPlatformInt()) {
		return 0, fmt.Errorf("%w: %v overflows int", ErrNotIntegral, f)
	}
	return int(f), nil
}

func maxPlatformInt() int {
	return int(^uint(0) >> 1)
}

func minPlatformInt() int {
	return -maxPlatformInt() - 1
}

// AsBool accepts only bools. Numeric or string conversions need a
// named transform in the mapper.
func AsBool(v any) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("%w: %T", ErrNotBool, v)
}

// ErrBatteryRange reports a battery percent outside 0-100.
var ErrBatteryRange = errors.New("battery percent outside 0-100")

// BatteryPercent coerces a mapped status value to a validated percent.
// Nil input means absent and returns nil without error. Out-of-range
// or non-numeric input returns an error; callers omit the battery and
// record diagnostics instead of clamping or fabricating a value.
func BatteryPercent(v any) (*int, error) {
	if v == nil {
		return nil, nil
	}
	i, err := AsInt(v)
	if err != nil {
		return nil, err
	}
	if i < 0 || i > 100 {
		return nil, fmt.Errorf("%w: %d", ErrBatteryRange, i)
	}
	return &i, nil
}
