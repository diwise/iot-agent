package milesight

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
)

func TestMilesightAM100Decoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(data_am100))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "24e124725c140542")

	blvl, ok := payload.Get[int](r, payload.BatteryLevelProperty)
	is.True(ok)
	is.Equal(blvl, 89)

	co2, ok := payload.Get[int](r, payload.CO2Property)
	is.True(ok)
	is.Equal(co2, 886)

	hlvl, ok := payload.Get[float32](r, payload.HumidityProperty)
	is.True(ok)
	is.Equal(hlvl, float32(29))

	templvl, ok := payload.Get[float64](r, payload.TemperatureProperty)
	is.True(ok)
	is.Equal(templvl, float64(22.3))
}

func TestMilesightEM500Decoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(data_em500))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "24e124126d154397")

	distance, ok := payload.Get[float64](r, payload.DistanceProperty)
	is.True(ok)
	is.Equal(distance, float64(5.0))
}

func TestMilesightDecoderEM400TLD(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(data_em400))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "24e124126d154397")

	distance, ok := payload.Get[float64](r, payload.DistanceProperty)
	is.True(ok)
	is.Equal(distance, float64(0.267))

	battery, ok := payload.Get[int](r, payload.BatteryLevelProperty)
	is.True(ok)
	is.Equal(battery, 98)

	position, ok := payload.Get[string](r, payload.PositionProperty)
	is.True(ok)
	is.Equal("normal", position)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const data_am100 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"AM103_1",
	"deviceProfileName":"Milesight AM100",
	"deviceProfileID":"c6a3467d-519d-4861-8e90-ba13a7b7c9ee",
	"devEUI":"24e124725c140542",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"AXVZA2ffAARoOgd9dgM=",
	"object":
	{
		"battery":89,
		"co2":886,
		"humidity":29,
		"temperature":22.3
	}
}`

const data_em500 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"EM500_UDL_1",
	"deviceProfileName":"Milesight EM500",
	"deviceProfileID":"f865a295-3d90-424e-967c-133c35d5594c",
	"devEUI":"24e124126d154397",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"A4KIEw==",
	"object":
	{
		"distance":5000
	}
}`

const data_em400 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"EM400_TLD",
	"deviceProfileName":"Milesight EM400",
	"deviceProfileID":"f865a295-3d90-424e-967c-133c35d5594c",
	"devEUI":"24e124126d154397",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"AXViA2f9/wSCCwEFAAA=",
	"object":
	{
		"battery": 98,
		"distance": 267,
		"position": "normal",
		"temperature": -0.3
	}
}`
