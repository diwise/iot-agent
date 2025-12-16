package talkpool

import (
	"fmt"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/matryer/is"
)

func TestPayload(t *testing.T) {
	is := is.New(t)

	for i, p := range testPayloads {

		t.Run(fmt.Sprintf("payload-%d", i), func(t *testing.T) {
			ue, err := facades.New("netmore")(nil, "payload", []byte(fmt.Sprintf(message, p)))
			is.NoErr(err)
			is.Equal(ue.DevEUI, "00138e0000007608")
			x, err := decodeOy1210Payload(ue.Payload.Data, 2)
			is.NoErr(err)
			fmt.Printf("Decoded payload %d: %+v\n", i, x)
		})
	}

}

func TestSinglePayload(t *testing.T) {
	is := is.New(t)

	ue, err := facades.New("netmore")(nil, "payload", []byte(fmt.Sprintf(message, testPayloads[0])))
	is.NoErr(err)
	is.Equal(ue.DevEUI, "00138e0000007608")
	x, err := decodeOy1210Payload(ue.Payload.Data, 2)
	is.NoErr(err)

	is.Equal(x.CO2, int(826))
	is.Equal(x.Humidity, 43.8)
	is.Equal(x.Temperature, 24.1)
}

var testPayloads = []string{"412B10033A", "4028F00321", "4128090319", "3D2AFE01AF", "3D2BE401AB"}

const message string = `[{
	"devEui":"00138e0000007608",
	"deviceName":"other",
	"sensorType":"other",
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
