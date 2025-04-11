package elsys

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"

	"github.com/matryer/is"
)

func TestElsysDigital1True(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := facades.New("servanet")("up", []byte(`{"data": "DQEaAA==", "fPort": 5, "devEui": "abc123", "object": {"digital": 1, "digital2": 0},  "timestamp": "2024-08-05T11:23:45.347949876Z", "deviceName": "abc123", "sensorType": "Elsys"}`))
	is.NoErr(err)
	p, err := decodePayload(ue.Payload.Data)
	is.NoErr(err)
	is.Equal(*p.DigitalInput, true)
}

func TestElsysDigital1True_lwm2m(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := facades.New("servanet")("up", []byte(`
	{
		"data": "DQEaAA==",
		"fPort": 5,
		"devEui": "abc123",
		"object": {
			"vdd": 3625,
			"digital": 1,
			"digital2": 0,
			"humidity": 100,
			"pressure": 1012.09,
			"temperature": 23.5
		},
		"timestamp": "2024-08-05T11:18:37.650212638Z",
		"deviceName": "braddmatare-3",
		"sensorType": "Elsys_codec"
	}
	`))
	is.NoErr(err)
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "abc123", payload, ue.Timestamp)
	is.NoErr(err)
	is.Equal(true, objects[3].(lwm2m.DigitalInput).DigitalInputState)
}

func TestElsysDigital1False(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := facades.New("servanet")("up", []byte(`{"data": "DQAaAA==", "fPort": 5, "devEui": "abc123", "object": {"digital": 1, "digital2": 0},  "timestamp": "2024-08-05T11:23:45.347949876Z", "deviceName": "abc123", "sensorType": "Elsys"}`))
	is.NoErr(err)
	p, err := decodePayload(ue.Payload.Data)
	is.NoErr(err)
	is.Equal(*p.DigitalInput, false)
}

func TestElsysDigital1False_lwm2m(t *testing.T) {
	is, _ := testSetup(t)
	ue, err := facades.New("servanet")("up", []byte(`{"data": "DQAaAA==", "fPort": 5, "devEui": "abc123", "object": {"digital": 1, "digital2": 0},  "timestamp": "2024-08-05T11:23:45.347949876Z", "deviceName": "abc123", "sensorType": "Elsys"}`))
	is.NoErr(err)
	p, err := decodePayload(ue.Payload.Data)
	is.NoErr(err)
	is.Equal(*p.DigitalInput, false)
	objects := convertToLwm2mObjects(context.Background(), "abc123", p, time.Now())
	is.Equal(false, objects[0].(lwm2m.DigitalInput).DigitalInputState)
}

func TestElsysCO2Decoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("servanet")("up", []byte(elsysCO2))
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "abc123", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(len(objects), 5)
}

func TestElsysTemperatureDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("servanet")("up", []byte(elsysTemp))
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "abc123", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(len(objects), 2)
}

func TestElsysPumpbrunnarDecoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, err := facades.New("netmore")("payload", []byte(elt2hp))
	is.NoErr(err)
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "abc123", payload, ue.Timestamp)
	is.NoErr(err)

	is.NoErr(err)
	is.Equal(len(objects), 4)
}

func TestDecodeElsysPayload(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")("payload", []byte(elt2hp))
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devId", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(len(objects), 4)

	singlePack := senml.Pack{}
	packs := lwm2m.ToPacks(objects)
	for _, p := range packs {
		err := p.Validate()
		is.NoErr(err)
		singlePack = append(singlePack, p...)
	}

	err = singlePack.Validate()
	is.NoErr(err)

	j, err := json.Marshal(singlePack)
	is.NoErr(err)

	is.Equal(string(j), `[{"bn":"devId/3303/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":7.5},{"bn":"devId/3304/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3304"},{"n":"5700","u":"%RH","v":84},{"bn":"devId/3/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3"},{"n":"7","u":"mV","v":3642},{"bn":"devId/3200/","bt":1698674257,"n":"0","vs":"urn:oma:lwm2m:ext:3200"},{"n":"5500","vb":false}]`)
}

func TestElsysElt2hpTrue(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")("payload", []byte(elt2hp_true))
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "abc123", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(len(objects), 4)
	is.True(objects[3].(lwm2m.DigitalInput).DigitalInputState)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const elt2hp_true string = `[{
	"devEui":"a81758fffe09ec03",
	"deviceName":"elt_2_hp",
	"sensorType":"elt_2_hp",
	"fPort":"5",
	"payload":"010096024e070e1e0d0114000fdcc01a00",
	"timestamp":"2023-10-30T13:57:37.868543Z",
	"rxInfo":{
		"gatewayId":"881",
		"rssi":-117,
		"snr":-17
	},
	"txInfo":{},
	"error":{}
}]`

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
