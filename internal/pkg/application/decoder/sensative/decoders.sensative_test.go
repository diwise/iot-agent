package sensative

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"

	"github.com/matryer/is"
)

func TestPresenceSensorReading(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.ChirpStack([]byte(livboj))

	objects, err := Decoder(context.Background(), "devID", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devID")
}

func TestPresenceSensorPeriodicCheckIn(t *testing.T) {
	is, _ := testSetup(t)
	ue := application.SensorEvent{}
	err := json.Unmarshal([]byte(livboj_checkin), &ue)
	is.NoErr(err)

	objects, err := Decoder(context.Background(), "devID", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devID")
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const livboj string = `
{
    "applicationID": "XYZ",
    "applicationName": "Livbojar",
    "deviceName": "Livboj",
    "deviceProfileName": "Sensative_Codec",
    "deviceProfileID": "8be301da",
	"devEUI": "3489573498573459",
    "rxInfo": [],
    "txInfo": {},
    "adr": true,
    "fCnt": 128,
    "fPort": 1,
    "data": "//8VAQ==",
    "object": {
        "closeProximityAlarm": {
            "value": true
        },
        "historySeqNr": 65535,
        "prevHistSeqNr": 65535
    }
}`

const livboj_checkin string = `{"devEui":"3489573498573459","deviceName":"Livboj","sensorType":"Sensative_Codec","fPort":1,"data":"//9uAxL8UAAAAAA=","object":{"buildId":{"id":51575888,"modified":false},"historySeqNr":65535,"prevHistSeqNr":65535},"timestamp":"2022-11-04T06:42:44.274490703Z","rxInfo":{"gatewayId":"fcc23dfffe2ee936","uplinkId":"23bab2ad-f4d0-4175-b09e-d1177dea44e0","rssi":-111,"snr":-8},"txInfo":{}}`
