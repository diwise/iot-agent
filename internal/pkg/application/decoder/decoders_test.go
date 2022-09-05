package decoder

import (
	"context"

	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestSenlabTBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := SenlabTBasicDecoder(context.Background(), []byte(senlabT), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.Timestamp, "2022-04-12T05:08:50.301732Z")
}

func TestElsysTemperatureDecoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := ElsysDecoder(context.Background(), []byte(elsysTemp), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.SensorType, "Elsys_Codec")
}

func TestElsysCO2Decoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := ElsysDecoder(context.Background(), []byte(elsysCO2), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.SensorType, "ELSYS")
}

func TestEnviotDecoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := EnviotDecoder(context.Background(), []byte(enviot), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.SensorType, "Enviot")
	is.Equal(len(r.Measurements), 4) // expected four measurements
}

func TestSenlabTBasicDecoderSensorReadingError(t *testing.T) {
	is, _ := testSetup(t)

	err := SenlabTBasicDecoder(context.Background(), []byte(senlabT_sensorReadingError), func(c context.Context, m Payload) error {
		return nil
	})

	is.True(err != nil)
}

func TestPresenceSensorReading(t *testing.T) {
	is, _ := testSetup(t)

	err := PresenceDecoder(context.Background(), []byte(livboj), func(ctx context.Context, p Payload) error {
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
	r := &Payload{}
	err := DefaultDecoder(context.Background(), []byte(elsysTemp), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})
	is.NoErr(err)
	is.True(r.DevEUI == "xxxxxxxxxxxxxx")
}

func TestQalcosonic_w1h_temp(t *testing.T) {
	is, _ := testSetup(t)

	payload := &Payload{}

	err := AxiomaWatermeteringDecoder(context.Background(), []byte(qalcosonic_w1h_temp), func(ctx context.Context, p Payload) error {
		payload = &p
		return nil
	})

	is.NoErr(err)
	is.Equal(payload.DevEUI, "116c52b4274f")
	is.Equal(payload.ValueOf("CurrentTime"), "2020-09-09T14:32:21+02:00")
	is.Equal(payload.ValueOf("CurrentVolume"), 302.57800000000003)
	is.Equal(payload.StatusCode, 0x7c)
}

func TestQalcosonic_w1h(t *testing.T) {
	is, _ := testSetup(t)

	payload := &Payload{}

	err := AxiomaWatermeteringDecoder(context.Background(), []byte(qalcosonic_w1h), func(ctx context.Context, p Payload) error {
		payload = &p
		return nil
	})

	is.NoErr(err)
	is.Equal(payload.DevEUI, "116c52b4274f")
	is.Equal(payload.ValueOf("CurrentTime"), "2022-08-25T09:41:28+02:00")
	is.Equal(payload.ValueOf("CurrentVolume"), 100.042)
	is.Equal(payload.StatusCode, 0)
}

func TestQalcosonic_w24h(t *testing.T) {
	is, _ := testSetup(t)

	payload := &Payload{}

	err := AxiomaWatermeteringDecoder(context.Background(), []byte(qalcosonic_w24h), func(ctx context.Context, p Payload) error {
		payload = &p
		return nil
	})

	is.NoErr(err)
	is.Equal(payload.DevEUI, "116c52b4274f")
	is.Equal(payload.ValueOf("CurrentTime"), "2022-09-02T15:40:16+02:00")
	is.Equal(payload.ValueOf("CurrentVolume"), 64.456)
	is.Equal(payload.StatusCode, 0)
}

func testSetup(t *testing.T) (*is.I, zerolog.Logger) {
	is := is.New(t)
	return is, zerolog.Logger{}
}

const senlabT string = `[{
	"devEui": "70b3d580a010f260",
	"sensorType": "tem_lab_14ns",
	"timestamp": "2022-04-12T05:08:50.301732Z",
	"payload": "01FE90619c10006A",
	"spreadingFactor": 12,
	"rssi": "-113",
	"snr": "-11.8",
	"gatewayIdentifier": 184,
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
	"spreadingFactor": 12,
	"rssi": "-113",
	"snr": "-11.8",
	"gatewayIdentifier": 184,
	"fPort": "3",
	"latitude": 57.806266,
	"longitude": 12.07727
}]`

/*const sensinglabstemp string = `{
	"devEUI":"70b3d580a010c589",
	"deviceName":"sk-senlab-temp-32",
	"rxInfo":[],"txInfo":{},
	"adr":true,"fCnt":1347,"fPort":3,
	"data":"AdCNeZwQARk=",
	"object":{"batlevel":81,"logValue":17.5625}
}`*/

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

const qalcosonic_w24h string = `
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
{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1h",
  "messageType": "payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "payload": "a827076300ca86010000a80363878401002300240023002a002400230023002400230051001e001d001b001d001c00",
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
}
`

const qalcosonic_w1h_temp string = `
{
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
}
`
