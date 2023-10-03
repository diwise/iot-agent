package decoder

import (
	"context"
	"io"

	"log/slog"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/matryer/is"

	. "github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func TestDefaultDecoder(t *testing.T) {
	is, _ := testSetup(t)
	var r Payload
	ue, _ := application.ChirpStack([]byte(data))
	err := DefaultDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})
	is.NoErr(err)
	is.Equal(r.DevEui(), "a81758fffe05e6fb")
}

func TestGetSlice(t *testing.T) {
	is := is.New(t)

	p, _ := New("test", time.Now().UTC(), S("s", struct{ S int }{1}), M("m", struct{ M int }{2}))
	s, _ := Get[int](p, "s")
	is.Equal(1, s)

	m, _ := GetSlice[struct{ M int }](p, "m")
	is.Equal(2, m[0].M)

	m2, _ := GetSlice[struct{ S int }](p, "s")
	is.Equal(1, m2[0].S)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const data string = `{
	"deviceName":"mcg-ers-co2-01",
	"deviceProfileName":"ELSYS",
	"deviceProfileID":"0b765672-274a-41eb-b1c5-bb2bec9d14e8",
	"devEUI":"a81758fffe05e6fb",
	"data":"AQDoAgwEAFoFAgYBqwcONA==",
	"object": {
		"co2":427,
		"humidity":12,
		"light":90,
		"motion":2,
		"temperature":23.2,
		"vdd":3636
	}
}`
