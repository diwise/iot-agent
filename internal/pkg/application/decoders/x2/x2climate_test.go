package x2

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
			x, err := DecodeX2Climate(ue.Payload.Data)
			is.NoErr(err)
			fmt.Printf("Decoded payload %d: %+v\n", i, x)
		})


	}

}

var testPayloads = []string{"031101149C1A00000D17002B088C2800000001", "031101149C1A00000D15002B088E2800000001", "031101149C1A00000D17002B08912800000001"}

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
