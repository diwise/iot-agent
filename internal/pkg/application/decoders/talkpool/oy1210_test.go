package talkpool

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
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

func TestUplingEvent(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	var ue types.Event

	err := json.Unmarshal([]byte(event), &ue)
	is.NoErr(err)
	is.Equal(ue.DevEUI, "70b3d5d720201643")
	x, err := decodeOy1210Payload(ue.Payload.Data, ue.Payload.FPort)
	is.NoErr(err)

	is.Equal(x.CO2, int(430))
	is.Equal(x.Humidity, 23.8)
	is.Equal(x.Temperature, 20.3)

	obj := convertToLwm2mObjects(ctx, ue.DevEUI, *x, ue.Timestamp)
	is.Equal(3, len(obj))
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

const event string = `
{
  "rx": {
    "rssi": -95,
    "loRaSNR": 5.5
  },
  "tx": {
    "dr": 5,
    "frequency": 868500000,
    "spreadingFactor": 7
  },
  "fCnt": 91163,
  "tags": {
    "customer": ["Higab"],
    "realestate": ["204"]
  },
  "devEUI": "70b3d5d720201643",
  "source": "sensor/??/payload",
  "payload": {
    "data": "Ph64Aa4=",
    "fPort": 2
  },
  "location": {
    "latitude": 0,
    "longitude": 0
  },
  "timestamp": "2026-01-30T10:42:22.396Z",
  "sensorType": "other"
}`
