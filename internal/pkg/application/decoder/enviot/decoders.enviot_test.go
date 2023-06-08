package enviot

import (
	"context"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestEnviotDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(enviot))
	err := Decoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	temp, _ := payload.Get[float64](r, payload.TemperatureProperty)
	is.Equal(temp, 11.5)
	humidity, _ := payload.Get[float32](r, payload.HumidityProperty)
	is.Equal(humidity, float32(85))
	batterylevel, _ := payload.Get[int](r, payload.BatteryLevelProperty)
	is.Equal(batterylevel, 86)
}

func testSetup(t *testing.T) (*is.I, zerolog.Logger) {
	is := is.New(t)
	return is, zerolog.Logger{}
}

const enviot string = `{
	"deviceProfileName":"Enviot",
	"devEUI":"10a52aaa84ffffff",
	"adr":false,
	"fCnt":56068,
	"fPort":1,
	"data":"VgAALuAAAAAAAAAABFtVAAGEtw==",
	"object":{
		"payload":{
			"battery":86,
			"distance":0,
			"fixangle":-60,
			"humidity":85,
			"pressure":995,
			"sensorStatus":0,
			"signalStrength":0,
			"snowHeight":0,
			"temperature":11.5,
			"vDistance":0
		}
	}
}`
