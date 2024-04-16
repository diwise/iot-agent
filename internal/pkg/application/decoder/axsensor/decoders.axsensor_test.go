package axsensor

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestAxsensor(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.Netmore([]byte(axsensor_input))

	objects, err := Decoder(context.Background(), "devId", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devId")

	is.Equal(len(objects), 3)

	device, _ := objects[2].(lwm2m.Device)
	is.Equal(*device.PowerSourceVoltage, 3488)

	temp, _ := objects[1].(lwm2m.Temperature)
	is.Equal(temp.SensorValue, 5.6)

	distance, _ := objects[0].(lwm2m.Distance)
	is.Equal(distance.SensorValue, float64(472))

}
func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const axsensor_input string = `
[{
      "devEui": "3731323054377916",
      "sensorType": "other",
      "timestamp": "2022-05-17T09:29:46.799874Z",
      "payload": "804024A4A00DA23800C80C0066FC",
      "spreadingFactor": "7",
      "rssi": "-45",
      "snr": "8.2",
      "gatewayIdentifier": "1378",
      "fPort": "2"
  }]`

//Payload med bara nivå:
// 804124	471,9

//804024A4A00DA23800C80C0066FC	level 472	batterispänning 3,49	temperatur 5.6
