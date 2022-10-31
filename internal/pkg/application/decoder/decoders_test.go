package decoder

import (
	"context"
	"fmt"

	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/matryer/is"
	"github.com/rs/zerolog"

	. "github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func TestSenlabTBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.Netmore([]byte(senlabT))
	err := SenlabTBasicDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	ts, _ := time.Parse(time.RFC3339Nano, "2022-04-12T05:08:50.301732Z")
	is.Equal(r.Timestamp(), ts)
}

func TestSenlabTBasicDecoderSensorReadingError(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.Netmore([]byte(senlabT_sensorReadingError))
	err := SenlabTBasicDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		return nil
	})

	is.True(err != nil)
}
func TestElsysTemperatureDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.ChirpStack([]byte(elsysTemp))
	err := ElsysDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "xxxxxxxxxxxxxx")
}

func TestElsysCO2Decoder(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.ChirpStack([]byte(elsysCO2))
	err := ElsysDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "a81758fffe05e6fb")
}

func TestEnviotDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.ChirpStack([]byte(enviot))
	err := EnviotDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	is.Equal(len(r.Measurements()), 4) // expected four measurements
}

func TestSensefarmBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.Netmore([]byte(sensefarm))
	err := SensefarmBasicDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})

	is.NoErr(err)
	ts, _ := time.Parse(time.RFC3339Nano, "2022-08-25T06:40:56.785171Z")
	is.Equal(r.Timestamp(), ts)
}

func TestPresenceSensorReading(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.ChirpStack([]byte(livboj))
	err := PresenceDecoder(context.Background(), ue, func(ctx context.Context, p Payload) error {
		return nil
	})

	is.NoErr(err)
}

func TestTimeStringConvert(t *testing.T) {
	is, _ := testSetup(t)

	tm, err := time.Parse(time.RFC3339, "1978-07-04T21:24:16.000000Z")

	min := tm.Unix()

	is.True(min == 268435456)
	is.NoErr(err)
}

func TestDefaultDecoder(t *testing.T) {
	is, _ := testSetup(t)
	var r Payload
	ue, _ := application.ChirpStack([]byte(elsysTemp))
	err := DefaultDecoder(context.Background(), ue, func(c context.Context, m Payload) error {
		r = m
		return nil
	})
	is.NoErr(err)
	is.Equal(r.DevEui(), "xxxxxxxxxxxxxx")
}

func TestQalcosonic_w1t(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	err := Qalcosonic_Auto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.True(r != nil)
	is.Equal("116c52b4274f", r.DevEui())
	is.Equal(302.57800000000003, r.ValueOf("CurrentVolume"))
	is.Equal(float64(25.78), r.ValueOf("temperature"))
	v, _ := Get[float64](r, "temperature")
	is.Equal(v, float64(25.78))
}

func TestQalcosonic_w1t_(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	err := Qalcosonic_w1t(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	ct, _ := time.Parse(time.RFC3339Nano, "2020-09-09T12:32:21Z")
	is.Equal(ct, r.ValueOf("CurrentTime"))
	is.Equal(r.ValueOf("CurrentVolume"), 302.57800000000003)
	is.Equal(r.Status().Code, 0x7c)
}

func TestQalcosonic_w1h(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1h))
	err := Qalcosonic_Auto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	is.Equal(r.ValueOf("CurrentTime"), toT("2019-07-22T11:37:50Z"))
	is.Equal(r.ValueOf("CurrentVolume"), 13.609)
	is.Equal(r.Status().Code, 48)
}

func TestQalcosonic_w1e(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload

	ue, _ := application.ChirpStack([]byte(qalcosonic_w1e))
	err := Qalcosonic_Auto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	is.Equal(r.ValueOf("CurrentTime"), toT("2022-09-02T13:40:16Z"))
	is.Equal(r.ValueOf("CurrentVolume"), 64.456)
	is.Equal(r.Status().Code, 0)
}

func TestQalcosonic_w1e_(t *testing.T) {
	is, _ := testSetup(t)
	var r Payload

	ue, _ := application.Netmore([]byte(qalcosonic_w1e_))
	err := Qalcosonic_w1e(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal("116c52b4274f", r.DevEui())
	is.Equal(toT("2020-05-29T07:51:59Z"), r.ValueOf("CurrentTime"))
}

func TestQalcosonic_w1e__(t *testing.T) {
	is, _ := testSetup(t)
	var r Payload

	ue, _ := application.Netmore([]byte(qalcosonic_w1e__))
	err := Qalcosonic_Auto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal("eroiu340958320409", r.DevEui())
	is.Equal(toT("2022-10-31T08:09:57Z"), r.ValueOf("CurrentTime"))
}

func TestQalcosonicStatusCodes(t *testing.T) {
	is, _ := testSetup(t)

	is.Equal("No error", getStatusMessage(0)[0])
	is.Equal("Power low", getStatusMessage(0x04)[0])
	is.Equal("Permanent error", getStatusMessage(0x08)[0])
	is.Equal("Temporary error", getStatusMessage(0x10)[0])
	is.Equal("Empty spool", getStatusMessage(0x10)[1])
	is.Equal("Leak", getStatusMessage(0x20)[0])
	is.Equal("Burst", getStatusMessage(0xA0)[0])
	is.Equal("Backflow", getStatusMessage(0x60)[0])
	is.Equal("Freeze", getStatusMessage(0x80)[0])

	is.Equal("Power low", getStatusMessage(0x0C)[0])
	is.Equal("Permanent error", getStatusMessage(0x0C)[1])

	is.Equal("Temporary error", getStatusMessage(0x10)[0])
	is.Equal("Empty spool", getStatusMessage(0x10)[1])

	is.Equal("Power low", getStatusMessage(0x14)[0])
	is.Equal("Temporary error", getStatusMessage(0x14)[1])
	is.Equal("Empty spool", getStatusMessage(0x14)[2])

	// ...

	is.Equal("Permanent error", getStatusMessage(0x18)[0])
	is.Equal("Temporary error", getStatusMessage(0x18)[1])
	is.Equal("Empty spool", getStatusMessage(0x18)[2])

	// ...

	is.Equal("Power low", getStatusMessage(0x3C)[0])
	is.Equal("Permanent error", getStatusMessage(0x3C)[1])
	is.Equal("Temporary error", getStatusMessage(0x3C)[2])
	is.Equal("Leak", getStatusMessage(0x3C)[3])

	// ...

	is.Equal("Power low", getStatusMessage(0xBC)[0])
	is.Equal("Permanent error", getStatusMessage(0xBC)[1])
	is.Equal("Temporary error", getStatusMessage(0xBC)[2])
	is.Equal("Burst", getStatusMessage(0xBC)[3])

	is.Equal("Unknown", getStatusMessage(0x02)[0])
}

func testSetup(t *testing.T) (*is.I, zerolog.Logger) {
	is := is.New(t)
	return is, zerolog.Logger{}
}

func toT(s any) time.Time {
	if str, ok := s.(string); ok {
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return t
		} else {
			panic(err)
		}
	} else {
		panic(fmt.Errorf("could not cast to string"))
	}
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

// payload ...0xFD14 = -46.75 = sensor reading error
const senlabT_sensorReadingError string = `[{
	"devEui": "70b3d580a010f260",
	"sensorType": "tem_lab_14ns",
	"timestamp": "2022-04-12T05:08:50.301732Z",
	"payload": "01FE90619c10FD14",
	"spreadingFactor": "12",
	"rssi": "-113",
	"snr": "-11.8",
	"gatewayIdentifier": "184",
	"fPort": "3",
	"latitude": 57.806266,
	"longitude": 12.07727
}]`

const sensefarm string = `[
	{
	  "devEui":"71b4d554600002b0",
	  "sensorType":"cube02",
	  "timestamp":"2022-08-25T06:40:56.785171Z",
	  "payload":"b006b800013008e4980000032fa80006990000043aa9000a08418a8bcc",
	  "spreadingFactor":"12",
	  "rssi":"-109",
	  "snr":"-2.5",
	  "gatewayIdentifier":"126",
	  "fPort":"2"
	}
  ]`

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

const enviot string = `{
	"deviceProfileName":"Enviot",
	"devEUI":"10a52aaa84ffffff",
	"adr":false,
	"fCnt":56068,
	"fPort":1,
	"data":"VgAALuAAAAAAAAAABFtVAAGEtw==",
	"object":{
		"payload":{
			"battery":86,
			"distance":0,
			"fixangle":-60,
			"humidity":85,
			"pressure":995,
			"sensorStatus":0,
			"signalStrength":0,
			"snowHeight":0,
			"temperature":11.5,
			"vDistance":0
		}
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

const qalcosonic_w1e_ string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1e",
  "messageType": "payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "payload": "011fbfd05e30cd0f0800d4879e41865c1b42470d7283b8201608fec181981dd007f3919460218247b631784c1c9e87b8e17600",
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
    "application": ["ambiductor_test"],
    "customer": ["customer"],
    "deviceType": ["w1e"],
    "serial": ["00000000"]
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
`

const qalcosonic_w1e string = `
{
  "applicationID": "2",
  "applicationName": "Watermetering",
  "deviceName": "e6c58aad",
  "deviceProfileName": "Axioma_Universal_Codec",
  "deviceProfileID": "72205a4d-a38a-4a0c-8bc8-116c52b4274f",
  "devEUI": "116c52b4274f",
  "rxInfo": [
    {
      "gatewayID": "f1861610fe6782f0",
      "uplinkID": "e6c58aad-7a14-42cb-82f0-f1861610fe67",
      "name": "SN-LGW-034",
      "time": "2022-09-02T13:45:28.605718289Z",
      "rssi": -113,
      "loRaSNR": -4.8,
      "location": {
        "latitude": 63.4,
        "longitude": 17.5,
        "altitude": 0
      }
    }
  ],
  "txInfo": { "frequency": 867100000, "dr": 3 },
  "adr": true,
  "fCnt": 1675,
  "fPort": 100,
  "data": "AcAHEmMAyPsAAAAAAHAAAAAAgAJwAIgAB8ACIAC0ABOABpABIAARwAUAADQAAAAAQAAA",
  "object": {
    "curDateTime": "2022-09-02 15:40:16",
    "curVol": 64456,
    "deltaVol": {
      "id1": 0,
      "id10": 11,
      "id11": 2,
      "id12": 45,
      "id13": 19,
      "id14": 26,
      "id15": 25,
      "id16": 8,
      "id17": 17,
      "id18": 23,
      "id19": 0,
      "id2": 0,
      "id20": 13,
      "id21": 0,
      "id22": 0,
      "id23": 4,
      "id3": 7,
      "id4": 0,
      "id5": 0,
      "id6": 10,
      "id7": 7,
      "id8": 34,
      "id9": 7
    },
    "frameVersion": 1,
    "statusCode": 0
  },
  "tags": { "Location": "UnSet", "SerialNo": "116c52b4274f" }
}
`
const qalcosonic_w1h string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1h",
  "messageType": "payload",
  "timestamp": "2019-07-27T11:37:50.834484Z",
  "payload": "0ea0355d302935000030b6345de7290000b800b900b800b800b800b900b800b800b800b800b800b800b900b900b900",
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
    "application": ["ambiductor_test"],
    "customer": ["customer"],
    "deviceType": ["w1e"],
    "serial": ["00000000"]
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
`

const qalcosonic_w1t string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1h_temp",
  "messageType": "payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "payload": "55cb585f7cf29d0400120ae0fe575f8a570400cd04cb04cc04cd04ca04c404c504c404f004e604dc04d604b9057905",
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
    "application": ["ambiductor_test"],
    "customer": ["customer"],
    "deviceType": ["w1e"],
    "serial": ["00000000"]
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
`

const qalcosonic_w1e__ string = `
[{
        "devEui": "eroiu340958320409",
        "sensorType": "qalcosonic_w1e",
        "messageType": "payload",
        "timestamp": "2022-10-31T08:17:08.124203Z",
        "payload": "d5825f63003090060090ad5e63518f060015001c001200110018000000000000000000000004000a00320027000c00",
        "fCntUp": 6732,
        "toa": null,
        "freq": 867900000,
        "batteryLevel": "255",
        "ack": false,
        "spreadingFactor": "8",
        "rssi": "-108",
        "snr": "-2",
        "gatewayIdentifier": "126",
        "fPort": "100",
        "tags": {
            "application": ["1_xyz_w1e_1"],
            "customer": ["xyz"],
            "deviceType": ["w1e"],
            "facilityID": [],
            "municipality": [],
            "serial": ["05014191"]
        },
        "gateways": [{
                "rssi": "-108",
                "snr": "-2",
                "gatewayIdentifier": "126",
                "antenna": 0
            }
        ]
    }
]`
