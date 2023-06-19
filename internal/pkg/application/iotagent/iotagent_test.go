package iotagent

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	iotcore "github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestSenlabTPayload(t *testing.T) {
	is, dmc, e, s := testSetup(t)

	agent := New(dmc, e, s).(*app)
	ue, _ := application.Netmore([]byte(senlabT))
	err := agent.HandleSensorEvent(context.Background(), ue)

	is.NoErr(err)
	is.True(len(e.SendCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].Value == 6.625)
}

func TestStripsPayload(t *testing.T) {
	is, dmc, e, s := testSetup(t)

	agent := New(dmc, e, s).(*app)
	ue, _ := application.Netmore([]byte(stripsPayload))
	err := agent.HandleSensorEvent(context.Background(), ue)

	is.NoErr(err)
	is.True(len(e.SendCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(pack[0].BaseName == "urn:oma:lwm2m:ext:3303")
}

func TestElsysPayload(t *testing.T) {
	is, dmc, e, s := testSetup(t)

	agent := New(dmc, e, s).(*app)
	ue, _ := application.ChirpStack([]byte(elsys))
	err := agent.HandleSensorEvent(context.Background(), ue)

	is.NoErr(err)
	is.True(len(e.SendCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].Value == 19.3)
}

func TestErsPayload(t *testing.T) {
	is, dmc, e, s := testSetup(t)

	agent := New(dmc, e, s).(*app)
	ue, _ := application.ChirpStack([]byte(ers))
	err := agent.HandleSensorEvent(context.Background(), ue)

	is.NoErr(err)
	is.True(len(e.SendCalls()) == 2) // expecting two calls since payload should produce measurement for both temperature and co2.

	tempPack := getPackFromSendCalls(e, 0) // the first call to send is for the temperature pack.
	is.True(tempPack[0].BaseName == "urn:oma:lwm2m:ext:3303")
	is.True(tempPack[1].Name == "5700")

	co2Pack := getPackFromSendCalls(e, 1) // the second call to send is for the co2 pack.

	is.True(co2Pack[0].BaseName == "urn:oma:lwm2m:ext:3428")
	is.True(co2Pack[1].Name == "17")
}

func TestPresencePayload(t *testing.T) {
	is, dmc, e, s := testSetup(t)

	agent := New(dmc, e, s).(*app)
	ue, _ := application.ChirpStack([]byte(livboj))
	err := agent.HandleSensorEvent(context.Background(), ue)

	is.NoErr(err)
	is.True(len(e.SendCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].BoolValue)
}

func getPackFromSendCalls(e *events.EventSenderMock, i int) senml.Pack {
	m := e.SendCalls()[i].M.(*iotcore.MessageReceived)
	return m.Pack
}

func testSetup(t *testing.T) (*is.I, *dmctest.DeviceManagementClientMock, *events.EventSenderMock, *storage.StorageMock) {
	is := is.New(t)
	dmc := &dmctest.DeviceManagementClientMock{
		FindDeviceFromDevEUIFunc: func(ctx context.Context, devEUI string) (client.Device, error) {

			types := []string{"urn:oma:lwm2m:ext:3303"}
			sensorType := "Elsys_Codec"

			if devEUI == "70b3d580a010f260" {
				sensorType = "tem_lab_14ns"
			} else if devEUI == "70b3d52c00019193" {
				sensorType = "strips_lora_ms_h"
			} else if devEUI == "a81758fffe05e6fb" {
				sensorType = "Elsys_Codec"
				types = []string{"urn:oma:lwm2m:ext:3303", "urn:oma:lwm2m:ext:3428"}
			} else if devEUI == "3489573498573459" {
				sensorType = "presence"
				types = []string{"urn:oma:lwm2m:ext:3302"}
			}

			res := &dmctest.DeviceMock{
				IDFunc:         func() string { return "internal-id-for-device" },
				SensorTypeFunc: func() string { return sensorType },
				TypesFunc:      func() []string { return types },
				IsActiveFunc:   func() bool { return true },
				TenantFunc:     func() string { return "default" },
			}

			return res, nil
		},
	}

	e := &events.EventSenderMock{
		SendFunc: func(ctx context.Context, m messaging.CommandMessage) error {
			return nil
		},
		PublishFunc: func(ctx context.Context, m messaging.TopicMessage) error {
			return nil
		},
	}

	s := &storage.StorageMock{
		AddFunc: func(ctx context.Context, id string, pack senml.Pack, timestamp time.Time) error { return nil },
		AddManyFunc: func(ctx context.Context, id string, packs []senml.Pack, timestamp time.Time) error {
			return nil
		},
	}

	return is, dmc, e, s
}

const senlabT string = `[{
	"devEui": "70b3d580a010f260",
	"sensorType": "tem_lab_14ns",
	"timestamp": "2022-04-12T05:08:50.301732Z",
	"payload": "01FE90619c10006A",
	"spreadingFactor": "12",
	"rssi": "-113",
	"snr": "-11.8",
	"gatewayIdentifier": "184",
	"fPort": "3",
	"latitude": 57.806266,
	"longitude": 12.07727
}]`

const stripsPayload string = `[{"devEui":"70b3d52c00019193","sensorType":"strips_lora_ms_h","timestamp":"2022-04-21T09:33:40.713643Z","payload":"ffff01590200d90400d4063c07000008000009000a01","spreadingFactor":"10","rssi":"-108","snr":"-3","gatewayIdentifier":"824","fPort":"1"}]`

const elsys string = `{
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

const ers string = `
{
    "deviceName": "mcg-ers-co2-01",
    "deviceProfileName": "ELSYS",
    "deviceProfileID": "0b765672-274a-41eb-b1c5-bb2bec9d14e8",
    "devEUI": "a81758fffe05e6fb",
    "data": "AQDuAhYEALIFAgYBxAcONA==",
    "object": {
        "co2": 452,
        "humidity": 22,
        "light": 178,
        "motion": 2,
        "temperature": 23.8,
        "vdd": 3636
    }
}`

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
