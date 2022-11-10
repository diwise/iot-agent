package conversion

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/matryer/is"
)

func TestThatTemperatureDecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Temperature(22.2))

	msg, err := Temperature(ctx, "internalID", p)

	is.NoErr(err)
	is.Equal(22.2, *msg[1].Value)
}

func TestThatCO2DecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.CO2(22))

	msg, err := AirQuality(ctx, "internalID", p)

	is.NoErr(err)
	is.Equal(float64(22), *msg[1].Value)
}

func TestThatPresenceDecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Presence(true))

	msg, err := Presence(ctx, "internalID", p)

	is.NoErr(err)
	is.True(*msg[1].BoolValue)
}

func TestThatWatermeterDecodesW1hValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1h))
	decoder.QalcosonicW1h(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	msg, err := Watermeter(ctx, "deviceID", p)

	is.NoErr(err)
	is.Equal("urn:oma:lwm2m:ext:3424", msg[0].BaseName)

	is.Equal("LogVolume", msg[1].Name)
	is.Equal("CurrentDateTime", msg[2].Name)
	is.Equal("LogDateTime", msg[3].Name)
	is.Equal("DeltaVolume", msg[4].Name)

	is.Equal(528.333, *msg[1].Value)
	is.Equal("2020-05-29T07:51:59Z", msg[2].StringValue)
	is.Equal("2020-05-29T01:00:00Z", msg[3].StringValue)
	is.Equal(2.004, *msg[4].Value)
	is.Equal(528.333+2.004, *msg[4].Sum)
	is.Equal(int64(msg[4].Time), time.Unix(int64(msg[3].Time), 0).Add(1*time.Hour).Unix())
}

func TestThatWatermeterDecodesW1eValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1e))
	decoder.QalcosonicW1e(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	msg, err := Watermeter(ctx, "deviceID", p)

	is.NoErr(err)
	is.Equal("urn:oma:lwm2m:ext:3424", msg[0].BaseName)

	is.Equal("CurrentVolume", msg[1].Name)
	is.Equal("LogVolume", msg[2].Name)
	is.Equal("CurrentDateTime", msg[3].Name)
	is.Equal("LogDateTime", msg[4].Name)
	is.Equal("DeltaVolume", msg[5].Name)

	is.Equal(13.609, *msg[1].Value)
	is.Equal(10.727, *msg[2].Value)
	is.Equal("2019-07-22T11:37:50Z", msg[3].StringValue)
	is.Equal("2019-07-21T19:00:00Z", msg[4].StringValue)
	is.Equal(0.184, *msg[5].Value)
	is.Equal(int64(msg[5].Time), time.Unix(int64(msg[4].Time), 0).Add(1*time.Hour).Unix())
}

func TestThatWatermeterDecodesW1tValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	decoder.QalcosonicW1t(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	msg, err := Watermeter(ctx, "deviceID", p)

	is.NoErr(err)
	is.Equal("urn:oma:lwm2m:ext:3424", msg[0].BaseName)

	is.Equal("CurrentVolume", msg[1].Name)
	is.Equal("LogVolume", msg[2].Name)
	is.Equal("CurrentDateTime", msg[3].Name)
	is.Equal("LogDateTime", msg[4].Name)
	is.Equal("Temperature", msg[5].Name)
	is.Equal("DeltaVolume", msg[6].Name)

	is.Equal(302.578, *msg[1].Value)
	is.Equal(284.554, *msg[2].Value)
	is.Equal("2020-09-09T12:32:21Z", msg[3].StringValue)
	is.Equal("2020-09-08T22:00:00Z", msg[4].StringValue)
	is.Equal(1.229, *msg[6].Value)
	is.Equal(int64(msg[6].Time), time.Unix(int64(msg[4].Time), 0).Add(1*time.Hour).Unix())
}

func mcmTestSetup(t *testing.T) (*is.I, context.Context) {
	ctx, _ := logging.NewLogger(context.Background(), "test", "")
	return is.New(t), ctx
}

func toT(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	} else {
		panic(err)
	}
}

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
