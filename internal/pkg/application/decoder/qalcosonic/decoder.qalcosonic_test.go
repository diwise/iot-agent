package qalcosonic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestQalcosonic_w1t(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.True(r != nil)
	is.Equal("116c52b4274f", r.DevEui())
	temp, _ := payload.Get[float64](r, payload.TemperatureProperty)
	is.Equal(float64(2578), temp)
	timestamp, _ := payload.Get[time.Time](r, payload.TimestampProperty)
	is.Equal(timestamp, toT("2020-09-09T12:32:21Z"))            // time for reading
	is.Equal(r.Timestamp(), toT("2022-08-25T07:35:21.834484Z")) // time from gateway
	volumes, _ := payload.GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](r, payload.VolumeProperty)
	is.Equal(16, len(volumes))
	is.Equal(float64(0), volumes[0].Volume)
	is.Equal(float64(284554), volumes[0].Cumulated)
	is.Equal(float64(volumes[0].Cumulated+volumes[1].Volume), volumes[1].Cumulated)
	is.Equal(volumes[0].Time, toT("2020-09-08T22:00:00Z"))
}

func TestQalcosonic_w1h(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1h))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	is.Equal(r.Status().Code, 48)
	timestamp, _ := payload.Get[time.Time](r, payload.TimestampProperty)
	is.Equal(timestamp, toT("2020-05-29T07:51:59Z")) // time for reading
	volumes, _ := payload.GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](r, payload.VolumeProperty)
	is.Equal(24, len(volumes))
	is.Equal(float64(0), volumes[0].Volume)
	is.Equal(float64(528333), volumes[0].Cumulated)
	is.Equal(float64(volumes[0].Cumulated+volumes[1].Volume), volumes[1].Cumulated)
	is.Equal(volumes[0].Time, toT("2020-05-28T01:00:00Z"))
}

func TestQalcosonic_w1e(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload

	ue, _ := application.Netmore([]byte(qalcosonic_w1e))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	is.Equal(r.DevEui(), "116c52b4274f")
	is.Equal(r.Status().Code, 0x30)
	timestamp, _ := payload.Get[time.Time](r, payload.TimestampProperty)
	is.Equal(timestamp, toT("2019-07-22T11:37:50Z")) // time for reading
	volumes, _ := payload.GetSlice[struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}](r, payload.VolumeProperty)
	is.Equal(17, len(volumes))
	is.Equal(float64(0), volumes[0].Volume)
	is.Equal(float64(10727), volumes[0].Cumulated)
	is.Equal(float64(volumes[0].Cumulated+volumes[1].Volume), volumes[1].Cumulated)
	is.Equal(volumes[0].Time, toT("2019-07-21T19:00:00Z"))
}

func TestQalcosonicAlarmMessage(t *testing.T) {
	is, _ := testSetup(t)

	var r payload.Payload

	ue, _ := application.Netmore([]byte(qalcosonicAlarmPacket))
	err := Decoder(context.Background(), ue, func(ctx context.Context, p payload.Payload) error {
		r = p
		return nil
	})

	is.NoErr(err)
	timestamp, _ := payload.Get[time.Time](r, payload.TimestampProperty)
	is.Equal(timestamp, toT("2019-07-19T12:02:11Z")) // time for reading
	is.Equal(r.Status().Code, 136)
	is.Equal(r.Status().Messages[0], "Permanent error")
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

	is.Equal("Permanent error", getStatusMessage(0x88)[0])
	is.Equal("Freeze", getStatusMessage(0x88)[1])

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
		panic(errors.New("could not cast to string"))
	}
}

const qalcosonic_w1e string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1e",
  "messageType": "payload.Payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "Payload": "0ea0355d302935000054c0345de7290000b800b900b800b800b800b900b800b800b800b800b800b800b900b900b900",
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
  "messageType": "payload.Payload",
  "timestamp": "2019-07-27T11:37:50.834484Z",
  "Payload": "011fbfd05e30cd0f0800d4879e41865c1b42470d7283b8201608fec181981dd007f3919460218247b631784c1c9e87b8e17600",
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
  "messageType": "payload.Payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "Payload": "55cb585f7cf29d0400120ae0fe575f8a570400cd04cb04cc04cd04ca04c404c504c404f004e604dc04d604b9057905",
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
const qalcosonicAlarmPacket string = `
[{
  "devEui": "116c52b4274f",
  "sensorType": "qalcosonic_w1t",
  "messageType": "payload.Payload",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "Payload": "43b1315d88",
  "fCntUp": 1490,
  "toa": null,
  "freq": 867900000,
  "batteryLevel": "255",
  "ack": false,
  "spreadingFactor": "8",
  "rssi": "-115",
  "snr": "-1.8",
  "gatewayIdentifier": "000",
  "fPort": "103",
  "tags": {
    "application": ["ambiductor_test"],
    "customer": ["customer"],
    "deviceType": ["w1t"],
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
