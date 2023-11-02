package elsys

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
)

func TestElsysCO2Decoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(elsysCO2))
	err := Decoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "a81758fffe05e6fb")
}

func TestElsysTemperatureDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.ChirpStack([]byte(elsysTemp))
	err := Decoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "xxxxxxxxxxxxxx")
}

func TestElsysPumpbrunnarDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, err := application.Netmore([]byte(elt2hp))
	is.NoErr(err)
	err = Decoder(context.Background(), ue, func(c context.Context, m payload.Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "xxxxxxxxxx")
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const elt2hp string = `{
	"devEui":"a81758fffe09ec03",
	"deviceName":"elt_2_hp",
	"sensorType":"elt_2_hp",
	"fPort":"5",
	"data":"AQBvAkUHDicNABQADz0iGgA=",
	"timestamp":"2023-10-30T13:57:37.868543Z",
	"rxInfo":{
		"gatewayId":"881",
		"rssi":-117,
		"snr":-17
	},
	"txInfo":{},
	"error":{}
}`

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
