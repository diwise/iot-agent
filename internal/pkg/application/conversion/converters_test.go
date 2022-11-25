package conversion

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestThatTemperatureConvertsValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Temperature(22.2))

	var msg senml.Pack
	err := Temperature(ctx, "internalID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(22.2, *msg[1].Value)
}

func TestThatCO2DConvertsValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.CO2(22))

	var msg senml.Pack
	err := AirQuality(ctx, "internalID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(float64(22), *msg[1].Value)
}

func TestThatIlluminanceConvertsValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Light(22))

	var msg senml.Pack
	err := Illuminance(ctx, "internalID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(float64(22), *msg[1].Value)
}

func TestThatPresenceConvertsValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Presence(true))

	var msg senml.Pack
	err := Presence(ctx, "internalID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.True(*msg[1].BoolValue)
}

func TestThatWatermeterConvertsW1hValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1h))
	decoder.QalcosonicW1h(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	var msg senml.Pack
	err := WaterMeter(ctx, "deviceID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(WaterMeterURN, msg[0].BaseName)
	is.Equal(float64(528.333), *msg[1].Sum)
	is.Equal(toT("2020-05-28T01:00:00Z").Unix(), int64(msg[1].Time))
}

func TestThatWatermeterConvertsW1eValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1e))
	decoder.QalcosonicW1e(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	var msg senml.Pack
	err := WaterMeter(ctx, "deviceID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(WaterMeterURN, msg[0].BaseName)
	is.Equal(float64(10.727), *msg[1].Sum)
	is.Equal(toT("2019-07-21T19:00:00Z").Unix(), int64(msg[1].Time))
}

func TestThatWatermeterConvertsW1tValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	decoder.QalcosonicW1t(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	var msg senml.Pack
	err := WaterMeter(ctx, "deviceID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(WaterMeterURN, msg[0].BaseName)
	is.Equal(float64(284.554), *msg[1].Sum)
	is.Equal(toT("2020-09-08T22:00:00Z").Unix(), int64(msg[1].Time))
}

func TestThatSensefarmConvertsPressureAndConductivity(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	var r payload.Payload
	ue, _ := application.Netmore([]byte(sensefarm))
	decoder.SensefarmBasicDecoder(ctx, ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	var pressure senml.Pack
	err := Pressure(ctx, "deviceID", r, func(p senml.Pack) error {
		pressure = p
		return nil
	})

	is.NoErr(err)

	var conductivity senml.Pack
	err = Conductivity(ctx, "deviceID", r, func(p senml.Pack) error {
		conductivity = p
		return nil
	})

	is.NoErr(err)

	is.Equal(pressure[0].BaseName, PressureURN)
	is.Equal(pressure[0].Time, float64(0))
	is.Equal(pressure[0].StringValue, "deviceID")
	is.Equal(pressure[1].Name, "5700")
	is.Equal(*pressure[1].Value, float64(6))

	is.Equal(conductivity[0].BaseName, ConductivityURN)
	is.Equal(conductivity[0].Time, float64(0))
	is.Equal(conductivity[0].StringValue, "deviceID")

	is.Equal(conductivity[1].Name, "5700")
	is.Equal(*conductivity[1].Value, float64(0.001226993865030675))
}

func TestThatHumidityConvertsValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Humidity(22))

	var msg senml.Pack
	err := Humidity(ctx, "internalID", p, func(p senml.Pack) error {
		msg = p
		return nil
	})

	is.NoErr(err)
	is.Equal(float64(22), *msg[1].Value)
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
