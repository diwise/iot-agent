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

	fillLevel, _ := objects[0].(lwm2m.FillingLevel)
	expectedPercentage := float64(33.714290)
	is.Equal(*fillLevel.ActualFillingPercentage, expectedPercentage)
	is.Equal(*fillLevel.ActualFillingLevel, int64(47))
}

func TestAxsensor2(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.Netmore([]byte(input2))

	objects, err := Decoder(context.Background(), "devId", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devId")
	is.Equal(len(objects), 3)

	fillingLevel, _ := objects[0].(lwm2m.FillingLevel)
	expectedPercentage := float64(0.14286)
	is.Equal(*fillingLevel.ActualFillingPercentage, expectedPercentage)
	is.Equal(*fillingLevel.ActualFillingLevel, int64(0))

	pressure, _ := objects[1].(lwm2m.Pressure)
	is.Equal(pressure.SensorValue, float64(100500))

	temp, _ := objects[2].(lwm2m.Temperature)
	is.Equal(temp.SensorValue, 16.0)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const axsensor_input string = `[{
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

const input2 string = `[{
	"devEui":"363536305b398e11",
	"sensorType":"other",
	"messageType":"payload",
	"timestamp":"2024-04-25T09:15:38.869474Z",
	"payload":"80a336a1ed03a2a000a3e301c8f9ff4f02",
	"fCntUp":3505,
	"toa":null,
	"freq":868100000,
	"batteryLevel":"0",
	"ack":false,
	"spreadingFactor":"12",
	"dr":0,
	"rssi":"-122",
	"snr":"-3",
	"gatewayIdentifier":"640",
	"fPort":"2"
}]`
