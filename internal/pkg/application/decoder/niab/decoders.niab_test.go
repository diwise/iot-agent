package niab

import (
	"testing"

	"github.com/matryer/is"
)

func TestNIAB(t *testing.T) {
	is := is.New(t)

	payload, err := decode([]byte{0xcc, 0x0f, 0x03, 0xc5})
	is.NoErr(err)

	is.Equal(payload.Battery, 80)
	is.Equal(payload.Temperature, 15.0)
	is.Equal(*payload.Distance, 0.965)
}

func TestNIABCanReportMinusTemperatures(t *testing.T) {
	is := is.New(t)

	payload, err := decode([]byte{0xcc, 0xF0, 0x03, 0xc5})
	is.NoErr(err)

	is.Equal(payload.Temperature, -15.0)
}

func TestNIABIgnoresReadErrors(t *testing.T) {
	is := is.New(t)

	_, err := decode([]byte{0xcc, 0x0f, 0xff, 0xff})
	is.True(err != nil)
}

func TestNIABIgnoresTruncatedData(t *testing.T) {
	is := is.New(t)

	_, err := decode([]byte{0xcc, 0x0f})
	is.True(err != nil)
}

func TestNIABIgnoresTooLongData(t *testing.T) {
	is := is.New(t)

	_, err := decode([]byte{0xcc, 0x0f, 0xff, 0xff, 0xff, 0xff})
	is.True(err != nil)
}
