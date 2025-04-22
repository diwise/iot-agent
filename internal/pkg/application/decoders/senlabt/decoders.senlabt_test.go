package senlabt

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestSenlabTBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(senlabT))

	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devID", payload, ue.Timestamp)
	is.NoErr(err)

	ts, _ := time.Parse(time.RFC3339Nano, "2022-04-12T05:08:50.301732Z")
	is.Equal(objects[0].Timestamp(), ts)
}

func TestSenlabTTempDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("servanet")(context.Background(), "up", []byte(senlabTemp))
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devID", payload, ue.Timestamp)
	is.NoErr(err)

	v, ok := objects[1].(lwm2m.Temperature)
	is.True(ok)
	is.Equal(v.SensorValue, float64(22.375))
}

func TestSenlabTBasicDecoderSensorReadingError(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(senlabT_sensorReadingError))

	_, err := Decoder(context.Background(), ue)

	is.True(err != nil)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const senlabT string = `[{
	"devEui": "70b3d580a010f260",
	"sensorType": "tem_lab_14ns",
	"timestamp": "2022-04-12T05:08:50.301732Z",
	"payload": "01FE90619c10006A",
	"spreadingFactor": "12",
	"rssi": "-113",
	"snr": "-11.8",
	"gatewayIdentifier": "184",
	"fPort": "3",
	"latitude": 57.806266,
	"longitude": 12.07727
}]`

const senlabTemp string = `{
	"devEUI": "70b3d580a010c5c3",
	"adr": true,
	"fCnt": 7299,
	"fPort": 3,
	"data": "AbaOFpwQAWY="
 }`

// payload ...0xFD14 = -46.75 = sensor reading error
const senlabT_sensorReadingError string = `[{
	"devEui": "70b3d580a010f260",
	"sensorType": "tem_lab_14ns",
	"timestamp": "2022-04-12T05:08:50.301732Z",
	"payload": "01FE90619c10FD14",
	"spreadingFactor": "12",
	"rssi": "-113",
	"snr": "-11.8",
	"gatewayIdentifier": "184",
	"fPort": "3",
	"latitude": 57.806266,
	"longitude": 12.07727
}]`
