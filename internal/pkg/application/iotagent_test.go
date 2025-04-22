package application

import (
	"context"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	iotcore "github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func TestSenlabTPayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default")
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(senlabT))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].Value == 6.625)
}

func TestStripsPayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(stripsPayload))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(pack[0].StringValue, "urn:oma:lwm2m:ext:3303")
}

func TestElt2HpPayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(elt2hp))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(pack[0].StringValue, "urn:oma:lwm2m:ext:3200")
}

func TestElsysPayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("servanet")(ctx, "up", []byte(elsys))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].Value == 19.3)
}

func TestElsysDigital1Payload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("servanet")(ctx, "up", []byte(`
	{
		"data": "DQEaAA==",
		"fPort": 5,
		"devEui": "aabbccddee",
		"timestamp": "2024-08-05T11:23:45.347949876Z",
		"deviceName": "braddmatare-3",
		"sensorType": "Elsys_codec",
		"object": {
        	"digital": 1,
        	"digital2": 0
    	}
	}
	`))
	err := agent.HandleSensorEvent(ctx, ue)
	is.NoErr(err)
}

func TestErsPayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("servanet")(ctx, "up", []byte(ers))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.Equal(len(e.SendCommandToCalls()), 2) // expecting three calls since payload should produce measurement for both temperature and co2 and more...

	tempPack := getPackFromSendCalls(e, 0) // the first call to send is for the temperature pack.
	is.Equal(tempPack[0].StringValue, "urn:oma:lwm2m:ext:3303")
	is.Equal(tempPack[1].Name, "5700")

	co2Pack := getPackFromSendCalls(e, 1) // the second call to send is for the co2 pack.

	is.Equal(co2Pack[0].StringValue, "urn:oma:lwm2m:ext:3428")
	is.Equal(co2Pack[1].Name, "17")
}

func TestPresencePayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("servanet")(ctx, "up", []byte(livboj))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].BoolValue)
}

func TestDistancePayload(t *testing.T) {
	is, dmc, e, ctx := testSetup(t)

	agent := New(dmc, e, storage.NewInMemory(), true, "default").(*app)
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(vegapuls))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(*pack[1].Value, 1.80952)
}

func TestDeterministicGuid(t *testing.T) {
	is := is.New(t)
	uuid1 := DeterministicGUID("inputstring")
	uuid2 := DeterministicGUID("inputstring")
	is.Equal(uuid1, uuid2)
}

func getPackFromSendCalls(e *messaging.MsgContextMock, i int) senml.Pack {
	sendCalls := e.SendCommandToCalls()
	cmd := sendCalls[i].Command
	m := cmd.(*iotcore.MessageReceived)
	return m.Pack()
}

func testSetup(t *testing.T) (*is.I, *dmctest.DeviceManagementClientMock, *messaging.MsgContextMock, context.Context) {
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
			} else if devEUI == "aabbccddee" {
				sensorType = "Elsys_Codec"
				types = []string{"urn:oma:lwm2m:ext:3200"}
			} else if devEUI == "3489573498573459" {
				sensorType = "presence"
				types = []string{"urn:oma:lwm2m:ext:3302"}
			} else if devEUI == "a81758fffe09ec03" {
				sensorType = "elt_2_hp"
				types = []string{"urn:oma:lwm2m:ext:3200"}
			} else if devEUI == "04c46100008f70e4" {
				sensorType = "vegapuls_air_41"
				types = []string{"urn:oma:lwm2m:ext:3330"}
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

	e := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error { return nil },
		SendCommandToFunc:  func(ctx context.Context, command messaging.Command, key string) error { return nil },
	}

	return is, dmc, e, context.Background()
}

const vegapuls string = `[{
	"devEui":"04c46100008f70e4",
	"sensorType":"vegapuls_air_41",
	"timestamp":"2024-04-23T09:47:59.915747Z",
	"payload":"02003fe79e6b2d6000d6b2",
	"spreadingFactor":"10",
	"dr":2,
	"rssi":"-103",
	"snr":"8",
	"gatewayIdentifier":"640",
	"fPort":"1"
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

const stripsPayload string = `
[{
        "devEui": "70b3d52c00019193",
        "sensorType": "strips_lora_ms_h",
        "timestamp": "2022-04-21T09:33:40.713643Z",
        "payload": "ffff01590200d90400d4063c07000008000009000a01",
        "spreadingFactor": "10",
        "rssi": "-108",
        "snr": "-3",
        "gatewayIdentifier": "824",
        "fPort": "1"
    }
]`

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
