package sensefarm

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestSensefarmBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(sensefarm))

	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devID", payload, ue.Timestamp)
	is.NoErr(err)

	packs := lwm2m.ToPacks(objects)

	is.Equal(6, len(packs))

	/*
		s, _ := payload.GetSlice[struct {
			Pressure int16
		}](r, payload.PressureProperty)
		is.Equal(s[0].Pressure, int16(6000))

		ohm, _ := payload.GetSlice[struct {
			Resistance int32
		}](r, payload.ResistanceProperty)
		is.Equal(ohm[0].Resistance, int32(815))
	*/
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
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
