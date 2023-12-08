package elsys

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/lwm2m"
	"github.com/matryer/is"
)

func TestElsysCO2Decoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.ChirpStack([]byte(elsysCO2))
	objects, err := Decoder(context.Background(), "devId", ue)
	is.NoErr(err)
	
	is.Equal(len(objects), 5)
}

func TestElsysTemperatureDecoder(t *testing.T) {
	is, _ := testSetup(t)
	
	ue, _ := application.ChirpStack([]byte(elsysTemp))
	objects, err := Decoder(context.Background(), "devId", ue)
	is.NoErr(err)

	is.Equal(len(objects), 2)
}

func TestElsysPumpbrunnarDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, err := application.Netmore([]byte(elt2hp))
	is.NoErr(err)
	objects, err := Decoder(context.Background(), "devId", ue)
	is.NoErr(err)


	is.NoErr(err)
	is.Equal(len(objects), 4)
}

func TestDecodeElsysPayload(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.Netmore([]byte(elt2hp))
	objects, err := Decoder(context.Background(), "devId", ue)
	is.NoErr(err)

	is.Equal(len(objects), 4)

	p := lwm2m.ToSinglePack(objects)
	err = p.Validate()
	is.NoErr(err)

	j, err := json.Marshal(p)
	is.NoErr(err)

	is.Equal(string(j), `[{"bn":"devId/3303/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":7.5},{"bn":"devId/3304/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3304"},{"n":"5700","u":"%RH","v":84},{"bn":"devId/3411/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3411"},{"n":"1","u":"%","v":0},{"n":"3","u":"V","v":3.642},{"bn":"devId/3200/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},{"n":"5500","vb":false}]`)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const elt2hp string = `[{
	"devEui":"a81758fffe09ec03",
	"deviceName":"elt_2_hp",
	"sensorType":"elt_2_hp",
	"fPort":"5",
	"payload":"01004b0254070e3a0d0014000f5bea1a00",
	"timestamp":"2023-10-30T13:57:37.868543Z",
	"rxInfo":{
		"gatewayId":"881",
		"rssi":-117,
		"snr":-17
	},
	"txInfo":{},
	"error":{}
}]`

const elt2hp_negTemp string = `[{
	"devEui":"a81758fffe09ec03",
	"deviceName":"elt_2_hp",
	"sensorType":"elt_2_hp",
	"fPort":"5",
	"payload":"01ffbd0253070e1d0d0014000f63491a00",
	"timestamp":"2023-10-30T13:57:37.868543Z",
	"rxInfo":{
		"gatewayId":"881",
		"rssi":-117,
		"snr":-17
	},
	"txInfo":{},
	"error":{}
}]`

const elsysTemp string = `{
	"applicationID": "8",
	"applicationName": "Water-Temperature",
	"deviceName": "sk-elt-temp-16",
	"deviceProfileName": "Elsys_Codec",
	"deviceProfileID": "xxxxxxxxxxxx",
	"devEUI": "xxxxxxxxxxxxxx",
	"rxInfo": [{
		"gatewayID": "xxxxxxxxxxx",
		"uplinkID": "xxxxxxxxxxx",
		"name": "SN-LGW-047",
		"time": "2022-03-28T12:40:40.653515637Z",
		"rssi": -105,
		"loRaSNR": 8.5,
		"location": {
			"latitude": 62.36956091265246,
			"longitude": 17.319844410529534,
			"altitude": 0
		}
	}],
	"txInfo": {
		"frequency": 867700000,
		"dr": 5
	},
	"adr": true,
	"fCnt": 10301,
	"fPort": 5,
	"data": "Bw2KDADB",
	"object": {
		"externalTemperature": 19.3,
		"vdd": 3466
	},
	"tags": {
		"Location": "Vangen"
	}
}`

const elsysCO2 string = `{
	"deviceName":"mcg-ers-co2-01",
	"deviceProfileName":"ELSYS",
	"deviceProfileID":"0b765672-274a-41eb-b1c5-bb2bec9d14e8",
	"devEUI":"a81758fffe05e6fb",
	"data":"AQDoAgwEAFoFAgYBqwcONA==",
	"object": {
		"co2":427,
		"humidity":12,
		"light":90,
		"motion":2,
		"temperature":23.2,
		"vdd":3636
	}
}`
