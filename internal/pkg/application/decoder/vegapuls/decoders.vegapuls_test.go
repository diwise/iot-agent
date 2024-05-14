package vegapuls

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestVegapulsSensorPacketIdentifier2(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, packetIdentifier2)))
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	b, _ := objects[0].(lwm2m.Device)
	is.Equal(*b.BatteryLevel, int(96))

	dist, _ := objects[1].(lwm2m.Distance)
	is.Equal(dist.SensorValue, float64(2.0))

	tmp, _ := objects[2].(lwm2m.Temperature)
	is.Equal(tmp.SensorValue, float64(20.9))
}

func TestVegapulsSensorPacketIdentifier8(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, packetIdentifier8)))
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	b, _ := objects[0].(lwm2m.Device)
	is.Equal(*b.BatteryLevel, int(36))

	dist, _ := objects[1].(lwm2m.Distance)
	is.Equal(dist.SensorValue, float64(1.27439))

	tmp, _ := objects[2].(lwm2m.Temperature)
	is.Equal(tmp.SensorValue, float64(26))
}

func TestVegapulsSensorConvertsFromInchesToMeters(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, testDataInFeet)))
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	dist, _ := objects[1].(lwm2m.Distance)
	is.Equal(dist.SensorValue, float64(0.03237)) // the value in inches is 1.27439, which should equal 0.03237 in metres
}

func TestVegapulsSensorConvertsFromFahrenheitToCelsius(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, testDataFahrenheit)))
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	temp, _ := objects[2].(lwm2m.Temperature)
	is.Equal(temp.SensorValue, float64(25.0))
}

func TestVegapulsSensorReturnsErrOnIncompletePayload(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, partialTestData)))
	is.NoErr(err)

	_, err = Decoder(context.Background(), "devid", ue)
	is.True(err != nil)
}

func TestVegapulsSensorReturnsErrOnUnknownPacketIdentifier(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, unknownPacketIdentifier)))
	is.NoErr(err)

	_, err = Decoder(context.Background(), "devid", ue)
	is.True(err != nil)
}

func TestVegapulsSensorPacketIdentifier12(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := application.Netmore([]byte(fmt.Sprintf(testData, packetIdentifier12)))
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	b, _ := objects[0].(lwm2m.Device)
	is.Equal(*b.BatteryLevel, int(99))

	dist, _ := objects[1].(lwm2m.Distance)
	is.Equal(dist.SensorValue, float64(1.87334))

	tmp, _ := objects[2].(lwm2m.Temperature)
	is.Equal(tmp.SensorValue, float64(21.4))
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
	"payload": "%s",
	"timestamp":"2024-02-28T11:21:59.626943Z",
	"rxInfo":{
		"gatewayId":"274",
		"rssi":-107,"snr":4
	},
	"txInfo":{},
	"error":{}
}]`

// this is a custom payload
const packetIdentifier2 string = "0200400000002d6000d1af"

// this payload is the default, and the example is taken from the vegapuls_air_41 documentation
const packetIdentifier8 string = "083FA31F152D2401042009"

const testDataInFeet string = "02003FA31F152F2400FA09"

const testDataFahrenheit string = "083FA31F152D2403022109"

const partialTestData string = "02003FA31F152D2400FA"

const unknownPacketIdentifier string = "05003FA31F152D2400FA"

const packetIdentifier12 string = "0c3fefc9712d222f222f42af05af296300d620b2"
