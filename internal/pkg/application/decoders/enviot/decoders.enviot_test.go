package enviot

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestEnviotDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("servanet")([]byte(enviot))
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devID", payload, ue.Timestamp)
	is.NoErr(err)

	temp, _ := objects[0].(lwm2m.Temperature)
	is.Equal(temp.SensorValue, float64(11.5))

	humidity, _ := objects[1].(lwm2m.Humidity)
	is.Equal(humidity.SensorValue, float64(85))

	battery, _ := objects[2].(lwm2m.Device)
	is.Equal(*battery.BatteryLevel, int(86))
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
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
