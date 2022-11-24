package decoder

import (
	"context"
	"encoding/json"
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
	temp, _ := Get[float64](r, "temperature")
	is.Equal(temp, 11.5)
	humidity, _ := Get[int](r, "humidity")
	is.Equal(humidity, 85)
	batterylevel, _ := Get[int](r, "batterylevel")
	is.Equal(batterylevel, 86)
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

	s := GetSlice[struct {
		Pressure int16
	}](r, "pressure")
	is.Equal(s[0].Pressure, int16(6))

	ohm := GetSlice[struct {
		Resistance int32
	}](r, "resistance")
	is.Equal(ohm[0].Resistance, int32(815))
}

func TestPresenceSensorReading(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.ChirpStack([]byte(livboj))

	var resultPayload Payload
	err := PresenceDecoder(context.Background(), ue, func(ctx context.Context, p Payload) error {
		resultPayload = p
		return nil
	})
	is.NoErr(err)

	_, ok := resultPayload.Get("Presence")
	is.True(ok)
}

func TestPresenceSensorPeriodicCheckIn(t *testing.T) {
	is, _ := testSetup(t)
	ue := application.SensorEvent{}
	err := json.Unmarshal([]byte(livboj_checkin), &ue)
	is.NoErr(err)

	var r Payload
	err = PresenceDecoder(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})
	is.NoErr(err)
	is.True(r != nil)
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
	err := QalcosonicAuto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.True(r != nil)
	is.Equal("116c52b4274f", r.DevEui())
	temp, _ := Get[float64](r, "temperature")
	is.Equal(float64(2578), temp)
	timestamp, _ := Get[time.Time](r, "timestamp")
	is.Equal(timestamp, toT("2020-09-09T12:32:21Z"))            // time for reading
	is.Equal(r.Timestamp(), toT("2022-08-25T07:35:21.834484Z")) // time from gateway
	volume := GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](r, "volume")
	is.Equal(16, len(volume))
	is.Equal(float64(0), volume[0].Volume)
	is.Equal(float64(284554), volume[0].Cumulated)
	is.Equal(float64(volume[0].Cumulated+volume[1].Volume), volume[1].Cumulated)
	is.Equal(volume[0].Time, toT("2020-09-08T22:00:00Z"))
}

func TestQalcosonic_w1h(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1h))
	err := QalcosonicAuto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	is.Equal(r.Status().Code, 48)
	timestamp, _ := Get[time.Time](r, "timestamp")
	is.Equal(timestamp, toT("2020-05-29T07:51:59Z")) // time for reading
	volume := GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](r, "volume")
	is.Equal(24, len(volume))
	is.Equal(float64(0), volume[0].Volume)
	is.Equal(float64(528333), volume[0].Cumulated)
	is.Equal(float64(volume[0].Cumulated+volume[1].Volume), volume[1].Cumulated)
	is.Equal(volume[0].Time, toT("2020-05-28T01:00:00Z"))
}

func TestQalcosonic_w1e(t *testing.T) {
	is, _ := testSetup(t)

	var r Payload

	ue, _ := application.Netmore([]byte(qalcosonic_w1e))
	err := QalcosonicAuto(context.Background(), ue, func(ctx context.Context, p Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	is.Equal(r.Status().Code, 0x30)
	timestamp, _ := Get[time.Time](r, "timestamp")
	is.Equal(timestamp, toT("2019-07-22T11:37:50Z")) // time for reading
	volume := GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](r, "volume")
	is.Equal(17, len(volume))
	is.Equal(float64(0), volume[0].Volume)
	is.Equal(float64(10727), volume[0].Cumulated)
	is.Equal(float64(volume[0].Cumulated+volume[1].Volume), volume[1].Cumulated)
	is.Equal(volume[0].Time, toT("2019-07-21T19:00:00Z"))
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

func TestGetSlice(t *testing.T) {
	is := is.New(t)

	p, _ := New("test", time.Now().UTC(), S("s", struct{ S int }{1}), M("m", struct{ M int }{1}))
	s, _ := Get[int](p, "s")
	is.Equal(1, s)

	m := GetSlice[struct{ M int }](p, "m")
	is.Equal(1, m[0].M)

	m2 := GetSlice[struct{ S int }](p, "s")
	is.Equal(1, m2[0].S)
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

const livboj_checkin string = `{"devEui":"3489573498573459","deviceName":"Livboj","sensorType":"Sensative_Codec","fPort":1,"data":"//9uAxL8UAAAAAA=","object":{"buildId":{"id":51575888,"modified":false},"historySeqNr":65535,"prevHistSeqNr":65535},"timestamp":"2022-11-04T06:42:44.274490703Z","rxInfo":{"gatewayId":"fcc23dfffe2ee936","uplinkId":"23bab2ad-f4d0-4175-b09e-d1177dea44e0","rssi":-111,"snr":-8},"txInfo":{}}`

const qalcosonic_w1e string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1e",
  "messageType": "payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "payload": "0ea0355d302935000054c0345de7290000b800b900b800b800b800b900b800b800b800b800b800b800b900b900b900",
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

const qalcosonic_w1h string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1h",
  "messageType": "payload",
  "timestamp": "2019-07-27T11:37:50.834484Z",
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

const qalcosonic_w1t string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1t",
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
