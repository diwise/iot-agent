package decoder

import (
	"context"
	"io"

	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/matryer/is"
)

func TestDefaultDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.ChirpStack([]byte(data))

	objects, err := DefaultDecoder(context.Background(), "devID", ue)
	is.NoErr(err)

	is.Equal(objects[0].ID(), "devID")
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
