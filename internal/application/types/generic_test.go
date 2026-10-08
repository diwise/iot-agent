package types

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/matryer/is"
)

func TestGenericPayloadLookupExactAndNested(t *testing.T) {
	is := is.New(t)

	p := GenericPayload{
		"temperature": 21.5,
		"payload": map[string]any{
			"snowHeight": 3.0,
		},
	}

	v, found, err := p.Lookup("temperature")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, 21.5)

	v, found, err = p.Lookup("payload.snowHeight")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, 3.0)
}

func TestGenericPayloadLookupCaseInsensitive(t *testing.T) {
	is := is.New(t)

	p := GenericPayload{"Temperature": 24.1}

	v, found, err := p.Lookup("temperature")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, 24.1)

	_, found, err = p.Lookup("humidity")
	is.NoErr(err)
	is.True(!found)
}

func TestGenericPayloadLookupAmbiguous(t *testing.T) {
	is := is.New(t)

	p := GenericPayload{
		"Temperature":  1.0,
		"TEMPERATURE":  2.0,
		"temperature ": 3.0,
	}

	// "TEMPERATURE" and "Temperature" both fold-match "temperature".
	_, _, err := p.Lookup("temperature")
	is.True(err != nil)
	is.True(errors.Is(err, ErrAmbiguousKey))
}

func TestGenericPayloadLookupEscapedDotAndIndex(t *testing.T) {
	is := is.New(t)

	p := GenericPayload{
		"a.b":  7.0,
		"list": []any{10.0, 20.0},
		"deep": map[string]any{
			"arr": []any{
				map[string]any{"v": 1.0},
			},
		},
	}

	v, found, err := p.Lookup(`a\.b`)
	is.NoErr(err)
	is.True(found)
	is.Equal(v, 7.0)

	v, found, err = p.Lookup("list[1]")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, 20.0)

	v, found, err = p.Lookup("deep.arr[0].v")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, 1.0)
}

func TestGenericPayloadLookupMissingNullAndMismatch(t *testing.T) {
	is := is.New(t)

	p := GenericPayload{
		"present": nil,
		"list":    []any{1.0},
		"scalar":  5.0,
	}

	// Explicit null is found with a nil value.
	v, found, err := p.Lookup("present")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, nil)

	// Missing key is not found.
	_, found, err = p.Lookup("absent")
	is.NoErr(err)
	is.True(!found)

	// Index into a scalar is not found.
	_, found, err = p.Lookup("scalar[0]")
	is.NoErr(err)
	is.True(!found)

	// Field into a scalar is not found.
	_, found, err = p.Lookup("scalar.x")
	is.NoErr(err)
	is.True(!found)

	// Out-of-range index is not found.
	_, found, err = p.Lookup("list[9]")
	is.NoErr(err)
	is.True(!found)

	// Null parent is not found.
	_, found, err = p.Lookup("present.x")
	is.NoErr(err)
	is.True(!found)
}

func TestGenericPayloadLookupNilMap(t *testing.T) {
	is := is.New(t)

	var p GenericPayload
	_, found, err := p.Lookup("a.b")
	is.NoErr(err)
	is.True(!found)
}

func TestGenericPayloadLookupInvalidPaths(t *testing.T) {
	is := is.New(t)

	for _, path := range []string{"", ".", "a.", ".a", "a..b", "a[1", "a[]", "a[x]", "a[-1]", `a\`} {
		_, _, err := GenericPayload{}.Lookup(path)
		is.True(err != nil)
		is.True(errors.Is(err, ErrInvalidPath))
	}
}

func TestAsFloatMatrix(t *testing.T) {
	is := is.New(t)

	ok := map[string]any{
		"float64":    1.5,
		"float32":    float32(2.5),
		"int":        -3,
		"int16":      int16(4),
		"uint8":      uint8(250),
		"jsonNumber": json.Number("12.5"),
	}
	for _, v := range ok {
		_, err := AsFloat(v)
		is.NoErr(err)
	}

	bad := map[string]any{
		"string": "1.5",
		"bool":   true,
		"nil":    nil,
		"map":    map[string]any{},
		"slice":  []any{},
		"nan":    math.NaN(),
		"inf":    math.Inf(1),
	}
	for _, v := range bad {
		_, err := AsFloat(v)
		is.True(err != nil)
		is.True(errors.Is(err, ErrNotNumeric))
	}
}

func TestAsIntMatrix(t *testing.T) {
	is := is.New(t)

	v, err := AsInt(42.0)
	is.NoErr(err)
	is.Equal(v, 42)

	v, err = AsInt(json.Number("42"))
	is.NoErr(err)
	is.Equal(v, 42)

	for _, bad := range []any{42.5, 1e19, math.Pow(2, 60), "42", true, nil, math.NaN(), json.Number("4.5")} {
		_, err := AsInt(bad)
		is.True(err != nil)
	}
}

func TestAsBoolStrict(t *testing.T) {
	is := is.New(t)

	v, err := AsBool(true)
	is.NoErr(err)
	is.True(v)

	v, err = AsBool(false)
	is.NoErr(err)
	is.True(!v)

	for _, bad := range []any{1, 0, "true", nil} {
		_, err := AsBool(bad)
		is.True(err != nil)
		is.True(errors.Is(err, ErrNotBool))
	}
}

func TestDecodedResultImplementsSensorPayload(t *testing.T) {
	is := is.New(t)

	var _ SensorPayload = DecodedResult{}
	var _ SensorPayload = &DecodedResult{}

	bat := 87
	d := DecodedResult{
		Data:     GenericPayload{"temperature": 21.5},
		Battery:  &bat,
		Code:     "E1",
		Messages: []string{"low battery"},
	}

	is.Equal(*d.BatteryLevel(), 87)

	code, messages := d.Error()
	is.Equal(code, "E1")
	is.Equal(messages, []string{"low battery"})

	empty := DecodedResult{}
	is.True(empty.BatteryLevel() == nil)
	code, messages = empty.Error()
	is.Equal(code, "")
	is.Equal(len(messages), 0)
}

func TestDecodedResultRawKeysDoNotCollide(t *testing.T) {
	is := is.New(t)

	bat := 55
	d := DecodedResult{
		Data: GenericPayload{
			"status": "raw-status",
			"error":  "raw-error",
			"data":   "raw-data",
		},
		Battery:  &bat,
		Code:     "E2",
		Messages: []string{"typed"},
	}

	// Wrapper status comes from typed fields, never from raw keys.
	is.Equal(*d.BatteryLevel(), 55)
	code, messages := d.Error()
	is.Equal(code, "E2")
	is.Equal(messages, []string{"typed"})

	// Raw keys stay intact in decoderdata.
	v, found, err := d.Data.Lookup("status")
	is.NoErr(err)
	is.True(found)
	is.Equal(v, "raw-status")
}

func TestDecodedResultJSONWritesDecoderDataOnly(t *testing.T) {
	is := is.New(t)

	bat := 87
	d := DecodedResult{
		Data:     GenericPayload{"temperature": 21.5},
		Mapping:  GenericPayload{"temperature": 21.5, "extra": true},
		Battery:  &bat,
		Code:     "E1",
		Messages: []string{"low battery"},
		Warnings: []string{"w"},
	}

	b, err := json.Marshal(d)
	is.NoErr(err)
	is.Equal(string(b), `{"temperature":21.5}`)

	var back DecodedResult
	is.NoErr(json.Unmarshal(b, &back))
	is.Equal(back.Data["temperature"], 21.5)
}
