package mapper

import (
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestGetTransformUnknown(t *testing.T) {
	is := is.New(t)

	_, err := GetTransform("nope", nil)
	is.True(err != nil)
	is.True(errors.Is(err, ErrUnknownTransform))
}

func TestIdentityTransform(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("identity", nil)
	is.NoErr(err)
	v, err := fn(1.5)
	is.NoErr(err)
	is.Equal(v, 1.5)

	_, err = GetTransform("identity", map[string]any{"x": 1})
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidTransformParams))
}

func TestScaleTransform(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("scale", map[string]any{"factor": 0.001})
	is.NoErr(err)
	v, err := fn(1400.0)
	is.NoErr(err)
	// Multiply-then-round differs from divide at ulp level, so scale
	// must not replace legacy-exact conversions such as v/1000.
	// Adapters own those computations. This locks the behavior.
	is.True(v != 1400.0/1000)

	fn, err = GetTransform("scale", map[string]any{"factor": 2.0, "offset": 1.0})
	is.NoErr(err)
	v, err = fn(3.0)
	is.NoErr(err)
	is.Equal(v, 7.0)

	// Defaults are factor 1 and offset 0.
	fn, err = GetTransform("scale", nil)
	is.NoErr(err)
	v, err = fn(3.0)
	is.NoErr(err)
	is.Equal(v, 3.0)

	_, err = GetTransform("scale", map[string]any{"factor": "x"})
	is.True(errors.Is(err, ErrInvalidTransformParams))

	_, err = GetTransform("scale", map[string]any{"bogus": 1})
	is.True(errors.Is(err, ErrInvalidTransformParams))

	_, err = fn("abc")
	is.True(err != nil)
}

func TestReciprocalTransform(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("reciprocal", nil)
	is.NoErr(err)
	v, err := fn(4.0)
	is.NoErr(err)
	is.Equal(v, 0.25)

	_, err = fn(0.0)
	is.True(err != nil)
}

func TestToIntTransform(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("to_int", nil)
	is.NoErr(err)

	// Go int() truncation toward zero.
	v, err := fn(87.9)
	is.NoErr(err)
	is.Equal(v, 87.0)

	v, err = fn(-87.9)
	is.NoErr(err)
	is.Equal(v, -87.0)

	_, err = fn(1e19)
	is.True(err != nil)

	_, err = fn("87")
	is.True(err != nil)
}

func TestEqualsTransform(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("equals", map[string]any{"value": 2})
	is.NoErr(err)
	v, err := fn(2.0)
	is.NoErr(err)
	is.Equal(v, true)
	v, err = fn(1.0)
	is.NoErr(err)
	is.Equal(v, false)

	fn, err = GetTransform("equals", map[string]any{"value": "open"})
	is.NoErr(err)
	v, err = fn("open")
	is.NoErr(err)
	is.Equal(v, true)

	_, err = GetTransform("equals", nil)
	is.True(errors.Is(err, ErrInvalidTransformParams))
	_, err = GetTransform("equals", map[string]any{})
	is.True(errors.Is(err, ErrInvalidTransformParams))
}

func TestMagnetOpenToBool(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("magnet_open_to_bool", nil)
	is.NoErr(err)
	v, err := fn("open")
	is.NoErr(err)
	is.Equal(v, true)
	v, err = fn("close")
	is.NoErr(err)
	is.Equal(v, false)

	_, err = fn("ajar")
	is.True(err != nil)
	_, err = fn(1.0)
	is.True(err != nil)
}

func TestVDDToSOCMatchesLegacyCurve(t *testing.T) {
	is := is.New(t)

	fn, err := GetTransform("vdd_to_soc", nil)
	is.NoErr(err)

	cases := map[int]int{
		0:    0,
		3000: 0,
		3299: 7,
		3300: 7,
		3350: 19,
		3417: 51,
		3500: 85,
		3599: 100,
		3600: 100,
		3636: 100,
	}
	for mv, want := range cases {
		v, err := fn(float64(mv))
		is.NoErr(err)
		is.Equal(v, want)
	}

	_, err = fn(33.5)
	is.True(err != nil)
	_, err = fn("3636")
	is.True(err != nil)
}
