package qalcosonic

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"

	"github.com/matryer/is"
)

func TestQalcosonic_w1t(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonic_w1t))
	p, ap, err := decodePayload(context.Background(), ue)

	is.NoErr(err)
	is.True(p != nil)
	is.True(ap == nil)

	is.Equal(*p.Temperature, uint16(2578))
	is.Equal(15, len(p.Deltas))
}

func TestQalcosonic_w1h(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonic_w1h))
	p, ap, err := decodePayload(context.Background(), ue)

	is.NoErr(err)
	is.True(ap == nil)

	is.Equal(24, len(p.Deltas))
	is.Equal(uint8(48), p.StatusCode)
}

func TestQalcosonic_w1e(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonic_w1e))
	p, ap, err := decodePayload(context.Background(), ue)

	is.NoErr(err)
	is.True(ap == nil)

	is.Equal(16, len(p.Deltas))
	is.Equal(uint8(48), p.StatusCode)
}

func TestQalcosonic_w1e_lwm2m(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonic_w1e))
	p, ap, err := decodePayload(context.Background(), ue)

	is.NoErr(err)
	is.True(ap == nil)

	is.Equal(float64(13609), p.CurrentVolume)

	objects := convertToLwm2mObjects(context.Background(), "", p, nil)
	is.Equal(17, len(objects))

	is.Equal(float64(13.609), objects[16].(lwm2m.WaterMeter).CumulatedWaterVolume)
}

func TestQalcosonicAlarmMessage(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonicAlarmPacket))
	p, ap, err := decodePayload(context.Background(), ue)

	is.NoErr(err)

	is.True(p == nil)
	is.True(ap.StatusCode == uint8(136))
}

func TestQalcosonicW1t_2(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := facades.New("servanet")(context.Background(), "up", []byte(qalcosonic_w1t_2))
	p, ap, err := decodePayload(context.Background(), ue)
	is.NoErr(err)
	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)
	objects, err := Converter(context.Background(), "devid", payload, ue.Timestamp)
	is.NoErr(err)

	is.NoErr(err)
	is.True(p != nil)
	is.True(ap == nil)

	is.Equal(*p.Temperature, uint16(915))
	is.Equal(15, len(p.Deltas))
	is.Equal(17, len(objects))
}

func TestDecode(t *testing.T) {
	t.Skip("check how this test should be changed to deal with the changes in Decoder")

	is, _ := testSetup(t)

	ue, _ := facades.New("netmore")(context.Background(), "payload", []byte(qalcosonic_w1t))
	payload, err := Decoder(t.Context(), ue)
	_, ok := err.(*types.DecoderErr)
	is.True(ok)

	objects, err := Converter(context.Background(), "devid", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(17, len(objects))

	singlePack := senml.Pack{}
	packs := lwm2m.ToPacks(objects)
	for _, p := range packs {
		err := p.Validate()
		is.NoErr(err)
		singlePack = append(singlePack, p...)
	}

	err = singlePack.Validate()
	is.NoErr(err)
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

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
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
const qalcosonic_w1t_2 string = `
{
  "devEui": "116c52b4274f",
  "timestamp": "2022-08-25T07:35:21.834484Z",
  "data": "uEJFZwDdiwYAkwPQdERnQYkGAEgAFgAjAEIA0wAZABsACgCMAA4ABwAZAAIACQA=",
  "fPort": 100
}
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
