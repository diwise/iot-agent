package senlabt

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestSenlabTBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.Netmore([]byte(senlabT))
	err := SenlabTBasicDecoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	ts, _ := time.Parse(time.RFC3339Nano, "2022-04-12T05:08:50.301732Z")
	is.Equal(r.Timestamp(), ts)
}

func TestSenlabTTempDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(senlabTemp))
	err := SenlabTBasicDecoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)

	v, ok := payload.Get[float64](r, payload.TemperatureProperty)
	is.True(ok)
	is.Equal(v, float64(22.375))
}

func TestSenlabTBasicDecoderSensorReadingError(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.Netmore([]byte(senlabT_sensorReadingError))
	err := SenlabTBasicDecoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		return nil
	})

	is.True(err != nil)
}

func testSetup(t *testing.T) (*is.I, zerolog.Logger) {
	is := is.New(t)
	return is, zerolog.Logger{}
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
