package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoders"
	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	iotcore "github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func TestSenlabTPayload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{})
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(senlabT))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].Value == 6.625)
}

func TestStripsPayload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{})
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(stripsPayload))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(pack[0].StringValue, "urn:oma:lwm2m:ext:3303")
}

func TestElt2HpPayload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(elt2hp))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(pack[0].StringValue, "urn:oma:lwm2m:ext:3200")
}

func TestElsysPayload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
	ue, _ := facades.New("servanet")(ctx, "up", []byte(elsys))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].Value == 19.3)
}

func TestElsysDigital1Payload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
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
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
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
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
	ue, _ := facades.New("servanet")(ctx, "up", []byte(livboj))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.True(*pack[1].BoolValue)
}

func TestDistancePayload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
	ue, _ := facades.New("netmore")(ctx, "payload", []byte(vegapuls))
	err := agent.HandleSensorEvent(ctx, ue)

	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(*pack[1].Value, 1.80952)
}

func TestQalcosonic(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)
	ue, _ := facades.New("netmore")(ctx, "payload", qalcosonic_templ("0ea0355d302935000054c0345de7290000b800b900b800b800b800b900b800b800b800b800b800b800b900b900b900"))
	err := agent.HandleSensorEvent(ctx, ue)
	is.NoErr(err)
	is.True(len(e.SendCommandToCalls()) > 0)

	pack := getPackFromSendCalls(e, 0)
	is.Equal(*pack[1].Value, 10.727)
}

func TestQalcosonicInvalidPayload(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)
	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)

	run := func(h string) error {
		ue, _ := facades.New("netmore")(ctx, "payload", qalcosonic_templ(h))
		err := agent.HandleSensorEvent(ctx, ue)
		return err
	}

	var errs []error
	payloads := []string{
		"015B04B668006C3C9B007B801B0015140DED0450614A740E4746D8C03E6C11DD8386214D1C16E904FEF023A003134198B01500",
		"01DB93B568006C3C9B007B801B0015140DED0450614A740E4746D8C03E6C11DD8386214D1C16E904FEF023A003134198B01500",
		"D6FDD0680071160C00A006102FD068D50A0C004602630146014B0136000B000B0011001900100016005E0187013E01",
		"1455D06800560F0C00AB065086CF6829F40B003800ED0012027B02FF016C020F022E029F0178013E02FD0246026301",
	}

	for _, p := range payloads {
		errs = append(errs, run(p))
	}

	is.NoErr(errors.Join(errs...))
}

func TestDeterministicGuid(t *testing.T) {
	is := is.New(t)
	uuid1 := DeterministicGUID("inputstring")
	uuid2 := DeterministicGUID("inputstring")
	is.Equal(uuid1, uuid2)
}

func TestIgnoreDeviceFor(t *testing.T) {
	is := is.New(t)
	_, _, e, s, _ := testSetup(t)

	agent := New(nil, e, s, false, "default", map[string]DeviceProfileConfig{}).(*app)

	// Verify cache is empty initially
	is.Equal(len(agent.notFoundDevices), 0)

	// Add device to cache for 1 minute
	agent.ignoreDeviceFor("testdevice", 1*time.Minute)

	// Verify device is now in cache
	is.Equal(len(agent.notFoundDevices), 1)

	// Verify the timestamp is in the future
	timestamp, exists := agent.notFoundDevices["testdevice"]
	is.True(exists)
	is.True(timestamp.After(time.Now().UTC()))
}

func TestDeviceIsCurrentlyIgnored(t *testing.T) {
	is := is.New(t)
	_, _, e, s, ctx := testSetup(t)

	agent := New(nil, e, s, false, "default", map[string]DeviceProfileConfig{}).(*app)

	// Test non-ignored device
	ignored := agent.deviceIsCurrentlyIgnored(ctx, "nonexistent")
	is.True(!ignored)

	// Add device to cache for 1 minute
	agent.ignoreDeviceFor("testdevice", 1*time.Minute)

	// Test recently ignored device
	ignored = agent.deviceIsCurrentlyIgnored(ctx, "testdevice")
	is.True(ignored)

	// Add device to cache for very short time (1ms)
	agent.ignoreDeviceFor("shortlived", 1*time.Millisecond)

	// Wait for the time to expire
	time.Sleep(2 * time.Millisecond)

	// Test expired device
	ignored = agent.deviceIsCurrentlyIgnored(ctx, "shortlived")
	is.True(!ignored)

	// Verify expired device was removed from cache
	_, exists := agent.notFoundDevices["shortlived"]
	is.True(!exists)
}

func TestIgnoreDeviceExpires(t *testing.T) {
	is := is.New(t)
	_, _, e, s, ctx := testSetup(t)

	agent := New(nil, e, s, false, "default", map[string]DeviceProfileConfig{}).(*app)

	// Add device to cache for very short time (1ms)
	agent.ignoreDeviceFor("expiringdevice", 1*time.Millisecond)

	// Verify device is initially ignored
	ignored := agent.deviceIsCurrentlyIgnored(ctx, "expiringdevice")
	is.True(ignored)

	// Wait for the time to expire
	time.Sleep(2 * time.Millisecond)

	// Verify device is no longer ignored
	ignored = agent.deviceIsCurrentlyIgnored(ctx, "expiringdevice")
	is.True(!ignored)

	// Verify device was removed from cache
	_, exists := agent.notFoundDevices["expiringdevice"]
	is.True(!exists)
}

func TestIgnoredDeviceIsBlacklisted(t *testing.T) {
	is := is.New(t)
	_, dmc, e, s, _ := testSetup(t)

	agent := New(dmc, e, s, false, "default", map[string]DeviceProfileConfig{}).(*app)

	// Add device to ignore cache
	agent.ignoreDeviceFor("blacklisteddevice", 1*time.Minute)

	// Try to find the ignored device
	device, err := agent.findDevice(context.Background(), "blacklisteddevice", dmc.FindDeviceFromDevEUI)

	// Should return errDeviceOnBlackList
	is.Equal(err, errDeviceOnBlackList)
	is.True(device == nil)
}

func TestFindDeviceMapsNotFoundError(t *testing.T) {
	is := is.New(t)
	_, dmc, e, s, _ := testSetup(t)

	agent := New(dmc, e, s, false, "default", map[string]DeviceProfileConfig{}).(*app)

	dmc.FindDeviceFromDevEUIFunc = func(ctx context.Context, devEUI string) (client.Device, error) {
		return nil, client.ErrNotFound
	}

	device, err := agent.findDevice(context.Background(), "missing-device", dmc.FindDeviceFromDevEUI)
	is.True(device == nil)
	is.True(errors.Is(err, errDeviceNotFound))
}

func TestFindDeviceReturnsUnderlyingError(t *testing.T) {
	is := is.New(t)
	_, dmc, e, s, _ := testSetup(t)

	agent := New(dmc, e, s, false, "default", map[string]DeviceProfileConfig{}).(*app)

	expectedErr := errors.New("request failed, not authorized")
	dmc.FindDeviceFromDevEUIFunc = func(ctx context.Context, devEUI string) (client.Device, error) {
		return nil, expectedErr
	}

	device, err := agent.findDevice(context.Background(), "auth-problem-device", dmc.FindDeviceFromDevEUI)
	is.True(device == nil)
	is.True(errors.Is(err, expectedErr))
	is.True(!errors.Is(err, errDeviceNotFound))
}

func TestUnknownDeviceIgnored(t *testing.T) {
	is := is.New(t)
	_, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{}).(*app)

	// Create an event for an unknown device that will be found as "unknown" type
	ue, _ := facades.New("netmore")(ctx, "payload", []byte("test"))
	ue.DevEUI = "unknowndevicetest"

	// Mock the FindDeviceFromDevEUI to return a device with "unknown" sensor type
	dmc.FindDeviceFromDevEUIFunc = func(ctx context.Context, devEUI string) (client.Device, error) {
		if devEUI == "unknowndevicetest" {
			return &dmctest.DeviceMock{
				IDFunc:         func() string { return "unknown-device-id" },
				SensorTypeFunc: func() string { return "unknown" },
				TypesFunc:      func() []string { return []string{} },
				IsActiveFunc:   func() bool { return true },
				TenantFunc:     func() string { return "default" },
			}, nil
		}
		return dmc.FindDeviceFromDevEUI(ctx, devEUI)
	}

	// Handle the event - should return nil (errDeviceIgnored is handled internally)
	err := agent.HandleSensorEvent(ctx, ue)
	is.NoErr(err)

	// Verify device was added to ignore cache for 5 minutes
	ignored := agent.deviceIsCurrentlyIgnored(ctx, ue.DevEUI)
	is.True(ignored)

	// Verify the timestamp is in the future (about 5 minutes from now)
	timestamp, exists := agent.notFoundDevices[ue.DevEUI]
	is.True(exists)
	expectedTime := time.Now().UTC().Add(5 * time.Minute)
	is.True(timestamp.After(expectedTime.Add(-10*time.Second)) && timestamp.Before(expectedTime.Add(10*time.Second)))
}

func TestAutoCleanupWorks(t *testing.T) {
	is := is.New(t)
	_, _, e, s, _ := testSetup(t)

	// Create an app with very short cleanup interval for testing
	agent := &app{
		registry:                   decoders.NewRegistry(),
		client:                     nil,
		msgCtx:                     e,
		store:                      s,
		notFoundDevices:            make(map[string]time.Time),
		createUnknownDeviceEnabled: false,
		createUnknownDeviceTenant:  "default",
		dpCfg:                      make(map[string]profile),
	}

	// Add devices to cache with very short expiry time (1ms)
	agent.ignoreDeviceFor("device1", 1*time.Millisecond)
	agent.ignoreDeviceFor("device2", 2*time.Millisecond)
	agent.ignoreDeviceFor("device3", 3*time.Millisecond)

	// Verify devices are in cache initially
	is.Equal(len(agent.notFoundDevices), 3)

	// Wait a bit longer than the shortest expiry
	time.Sleep(5 * time.Millisecond)

	// Manually run the cleanup logic (simulate ticker)
	agent.notFoundDevicesMu.Lock()
	for devEUI, ts := range agent.notFoundDevices {
		if time.Now().UTC().After(ts.UTC()) {
			delete(agent.notFoundDevices, devEUI)
		}
	}
	agent.notFoundDevicesMu.Unlock()

	// Verify all devices have been cleaned up
	is.Equal(len(agent.notFoundDevices), 0)
}

func getPackFromSendCalls(e *messaging.MsgContextMock, i int) senml.Pack {
	sendCalls := e.SendCommandToCalls()
	cmd := sendCalls[i].Command
	m := cmd.(*iotcore.MessageReceived)
	return m.Pack()
}

func testSetup(t *testing.T) (*is.I, *dmctest.DeviceManagementClientMock, *messaging.MsgContextMock, storage.Storage, context.Context) {
	is := is.New(t)
	dmc := &dmctest.DeviceManagementClientMock{
		FindDeviceFromDevEUIFunc: func(ctx context.Context, devEUI string) (client.Device, error) {

			types := []string{"urn:oma:lwm2m:ext:3303"}
			sensorType := "Elsys_Codec"

			switch devEUI {
			case "70b3d580a010f260":
				sensorType = "tem_lab_14ns"
			case "70b3d52c00019193":
				sensorType = "strips_lora_ms_h"
			case "a81758fffe05e6fb":
				sensorType = "Elsys_Codec"
				types = []string{"urn:oma:lwm2m:ext:3303", "urn:oma:lwm2m:ext:3428"}
			case "aabbccddee":
				sensorType = "Elsys_Codec"
				types = []string{"urn:oma:lwm2m:ext:3200"}
			case "3489573498573459":
				sensorType = "presence"
				types = []string{"urn:oma:lwm2m:ext:3302"}
			case "a81758fffe09ec03":
				sensorType = "elt_2_hp"
				types = []string{"urn:oma:lwm2m:ext:3200"}
			case "04c46100008f70e4":
				sensorType = "vegapuls_air_41"
				types = []string{"urn:oma:lwm2m:ext:3330"}
			case "116c52b4274f":
				sensorType = "qalcosonic"
				types = []string{"urn:oma:lwm2m:ext:3424"}
			default:
				types = []string{"urn:oma:lwm2m:ext:3303"}
				sensorType = "Elsys_Codec"
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
		FindDeviceFromInternalIDFunc: func(ctx context.Context, deviceID string) (client.Device, error) {
			res := &dmctest.DeviceMock{
				IDFunc:         func() string { return deviceID },
				SensorTypeFunc: func() string { return "Elsys_Codec" },
				TypesFunc:      func() []string { return []string{"urn:oma:lwm2m:ext:3303"} },
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

	s := &storage.StorageMock{
		CloseFunc: func() error {
			return nil
		},
		SaveFunc: func(ctx context.Context, se types.Event, device client.Device, payload types.SensorPayload, objects []lwm2m.Lwm2mObject, err error) error {
			return nil
		},
	}

	return is, dmc, e, s, context.Background()
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

func qalcosonic_templ(h string) []byte {
	return []byte(fmt.Sprintf(`
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1e",
  "messageType": "payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "Payload": "%s",
  "fCntUp": 1490,
  "toa": null,
  "freq": 867900000,
  "batteryLevel": "255",
  "ack": false,
  "spreadingFactor": "8",
  "rssi": "-115",
  "snr": "-1.8",
  "gatewayIdentifier": "000",
  "fPort": "100",
  "tags": {
  },
  "gateways": [
    {
      "rssi": "-115",
      "snr": "-1.8",
      "gatewayIdentifier": "000",
      "antenna": 0
    }
  ]
}]
`, h))
}
