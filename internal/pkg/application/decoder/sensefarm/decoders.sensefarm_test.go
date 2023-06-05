package sensefarm

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestSensefarmBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.Netmore([]byte(sensefarm))
	err := BasicDecoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	ts, _ := time.Parse(time.RFC3339Nano, "2022-08-25T06:40:56.785171Z")
	is.Equal(r.Timestamp(), ts)

	s, _ := payload.GetSlice[struct {
		Pressure int16
	}](r, payload.PressureProperty)
	is.Equal(s[0].Pressure, int16(6000))

	ohm, _ := payload.GetSlice[struct {
		Resistance int32
	}](r, payload.ResistanceProperty)
	is.Equal(ohm[0].Resistance, int32(815))
}

func testSetup(t *testing.T) (*is.I, zerolog.Logger) {
	is := is.New(t)
	return is, zerolog.Logger{}
}

const sensefarm string = `[
	{
	  "devEui":"71b4d554600002b0",
	  "sensorType":"cube02",
	  "timestamp":"2022-08-25T06:40:56.785171Z",
	  "payload":"b006b800013008e4980000032fa80006990000043aa9000a08418a8bcc",
	  "spreadingFactor":"12",
	  "rssi":"-109",
	  "snr":"-2.5",
	  "gatewayIdentifier":"126",
	  "fPort":"2"
	}
  ]`
