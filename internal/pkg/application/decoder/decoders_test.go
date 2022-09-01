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

/*func TestSenlabTTempDecoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := SenlabTBasicDecoder(context.Background(), []byte(sensinglabstemp), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.Timestamp, "2022-04-12T05:08:50.301732Z")
}*/

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

func TestSensefarmBasicDecoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := SensefarmBasicDecoder(context.Background(), []byte(sensefarm), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.Equal(r.Timestamp, "2022-08-25T06:40:56.785171Z")
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

func TestWatermeteringDecoder(t *testing.T) {
	is, _ := testSetup(t)

	r := &Payload{}

	err := WatermeteringDecoder(context.Background(), []byte(watermetering), func(c context.Context, m Payload) error {
		r = &m
		return nil
	})

	is.NoErr(err)
	is.True(r.DevEUI == "3489573498573459")
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

const sensefarm string = `[
	{
	  "devEui":"70b3d554600002b0",
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

//const anotherCO2 string = `{"deviceName":"mcg-ers-co2-01","deviceProfileName":"ELSYS","deviceProfileID":"0b765672-274a-41eb-b1c5-bb2bec9d14e8","devEUI":"a81758fffe05e6fb","data":"AQD5AhMEAa8FCgYCcQcONA==","object":{"co2":625,"humidity":19,"light":431,"motion":10,"temperature":24.9,"vdd":3636}}`

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

const watermetering string = `
{
  "applicationID": "2",
  "applicationName": "Watermetering",
  "deviceName": "05394167",
  "deviceProfileName": "Axioma_Universal_Codec",
  "deviceProfileID": "8be301da",
  "devEUI": "3489573498573459",
  "txInfo": { "frequency": 867100000, "dr": 0 },
  "adr": true,
  "fCnt": 182,
  "fPort": 100,
  "data": "//8VAQ==",
  "object": {
    "curDateTime": "2022-02-10 15:13:57",
    "curVol": 1009,
    "deltaVol": {
      "id1": 0,
      "id10": 13,
      "id11": 10,
      "id12": 2,
      "id13": 0,
      "id14": 1,
      "id15": 0,
      "id16": 5,
      "id17": 0,
      "id18": 0,
      "id19": 0,
      "id2": 8,
      "id20": 2,
      "id21": 0,
      "id22": 0,
      "id23": 0,
      "id3": 0,
      "id4": 0,
      "id5": 0,
      "id6": 0,
      "id7": 0,
      "id8": 5,
      "id9": 6
    },
    "frameVersion": 1,
    "statusCode": 0
  },
  "tags": { "Location": "UnSet", "SerialNo": "05394167" }
}
`
