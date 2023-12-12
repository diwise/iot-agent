package milesight

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestMilesightAM100Decoder(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.ChirpStack([]byte(data_am100))

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	b, _ := objects[0].(lwm2m.Battery)
	is.Equal(b.BatteryLevel, int(89))

	co2, _ := objects[1].(lwm2m.AirQuality)
	is.Equal(*co2.CO2, float64(886))

	h, _ := objects[2].(lwm2m.Humidity)
	is.Equal(h.SensorValue, float64(29))

	tmp, _ := objects[3].(lwm2m.Temperature)
	is.Equal(tmp.SensorValue, float64(22.3))
}

func TestMilesightEM500Decoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.ChirpStack([]byte(data_em500))

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	d, _ := objects[0].(lwm2m.Distance)
	is.Equal(d.SensorValue, float64(5.0))
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
