package milesight

import (
	"context"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestMilesightDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(data))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "24e124725c140542")
}

func testSetup(t *testing.T) (*is.I, zerolog.Logger) {
	is := is.New(t)
	return is, zerolog.Logger{}
}

const data string = `{
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
	},
	"tags":
	{
		"location":"599A",
		"mount":"wall"
	}
}`
