package vegapuls

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestVegapulsSensor(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(testData))
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	b, _ := objects[0].(lwm2m.Device)
	is.Equal(*b.BatteryLevel, int(36))

	tmp, _ := objects[2].(lwm2m.Temperature)
	is.Equal(tmp.SensorValue, float64(25))
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const testData string = `[{
	"devEui":"c4ac590000ccc60d",
	"deviceName":"vegapuls_air_41",
	"sensorType":"vegapuls_air_41",
	"fPort":"1",
	"payload":"02003FA31F152D2400FA09",
	"timestamp":"2024-02-28T11:21:59.626943Z",
	"rxInfo":{
		"gatewayId":"274",
		"rssi":-107,"snr":4
	},
	"txInfo":{},
	"error":{}
}]`
