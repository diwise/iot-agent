package storage

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/lwm2m"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/qalcosonic"
	"github.com/matryer/is"
)

func TestSQL(t *testing.T) {
	// start TimescaleDB using 'docker compose -f deployments/docker-compose.yaml up'
	// test will PASS if no DB is running

	is, s, ctx, err := testSetup(t)
	if err != nil {
		return
	}

	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	objects, err := qalcosonic.Decoder(context.Background(), "devID", ue)
	is.NoErr(err)

	packs := lwm2m.ToPacks(objects)

	err = s.Add(ctx, "devID", packs[0], time.Now())
	is.NoErr(err)

	storedPacks, err := s.GetMeasurements(ctx, "devID", "", time.Unix(0, 0), time.Now(), 1000)
	is.NoErr(err)

	if len(storedPacks) == 0 {
		t.Fail()
	}
}

func testSetup(t *testing.T) (*is.I, Storage, context.Context, error) {
	cfg := Config{
		host:     "localhost",
		user:     "diwise",
		password: "diwise",
		port:     "5432",
		dbname:   "diwise",
		sslmode:  "disable",
	}

	ctx := context.Background()

	s, err := Connect(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	_ = s.Initialize(ctx)

	is := is.New(t)

	return is, s, ctx, nil
}

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
