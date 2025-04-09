package sensative

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"

	"github.com/matryer/is"
)

func TestPresenceSensorReading(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := facades.ChirpStack([]byte(livboj))

	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devID", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(objects[0].ID(), "devID")
	is.Equal(objects[0].(lwm2m.Presence).DigitalInputState, true)
}

func TestPresenceSensorPeriodicCheckIn(t *testing.T) {
	is, _ := testSetup(t)
	ue := types.SensorEvent{}
	err := json.Unmarshal([]byte(livboj_checkin), &ue)
	is.NoErr(err)

	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devID", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(objects[0].ID(), "devID")
}

func TestDataErrSensorReading(t *testing.T) {
	is, _ := testSetup(t)
	ue := types.SensorEvent{}

	payload := []string{
		"//9uAxL8UAAAAAA=",
		"//9uAxL8UAUAAAA=",
		"//9uAxL8UPkAAAA=",
		"//9uAxL8UPwAAAA=",
	}

	for _, p := range payload {
		d := fmt.Sprintf(checkin, p)
		err := json.Unmarshal([]byte(d), &ue)
		is.NoErr(err)
		_, err = Decoder(context.Background(), ue)
		is.NoErr(err)
	}
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

const checkin string = `
{
  "data": "%s",
  "error": {},
  "fPort": 1,
  "devEui": "a4bc",
  "object": {
    "buildId": {
      "id": 51575888,
      "modified": false
    },
    "historySeqNr": 65535,
    "prevHistSeqNr": 65535
  },
  "rxInfo": {
    "snr": -14,
    "rssi": -127,
    "uplinkId": "00a2332a-cead-4974-a92f-8ee199747b1a",
    "gatewayId": "0016c001ff10c7f6"
  },
  "txInfo": {},
  "timestamp": "2024-11-12T07:03:37.187954862Z",
  "deviceName": "Livboj-05",
  "sensorType": "Sensative_Codec"
}`
