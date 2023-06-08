package storage

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/qalcosonic"
	"github.com/farshidtz/senml/v2"
	"github.com/rs/zerolog"
)

func TestSQL(t *testing.T) {
	// start TimescaleDB using 'docker compose -f deployments/docker-compose.yaml up'
	// test will PASS if no DB is running

	s, ctx, err := testSetup()
	if err != nil {
		return
	}

	var p payload.Payload
	ue, _ := application.Netmore([]byte(qalcosonic_w1t))
	qalcosonic.W1Decoder(context.Background(), ue, func(ctx context.Context, pp payload.Payload) error {
		p = pp
		return nil
	})

	var pack senml.Pack
	err = conversion.Watermeter(ctx, "deviceID", p, func(p senml.Pack) error {
		pack = p
		return nil
	})

	if err != nil {
		t.Error(err)
	}

	err = s.Add(ctx, "devID", pack, time.Now())
	if err != nil {
		t.Error(err)
	}

	packs, err := s.GetMeasurements(ctx, "devID")
	if err != nil {
		t.Error(err)
	}

	if len(packs) == 0 {
		t.Fail()
	}
}

func testSetup() (Storage, context.Context, error) {
	cfg := Config{
		host:     "localhost",
		user:     "diwise",
		password: "diwise",
		port:     "5432",
		dbname:   "diwise",
		sslmode:  "disable",
	}

	ctx := context.Background()

	s, err := Connect(ctx, zerolog.Logger{}, cfg)
	if err != nil {
		return nil, nil, err
	}

	_ = s.Initialize(ctx)

	return s, ctx, nil
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
