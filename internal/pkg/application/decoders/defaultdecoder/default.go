package defaultdecoder

import (
	"context"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	log := logging.GetFromContext(ctx)
	log.Info("default decoder used", slog.String("sensor_type", e.SensorType))

	return nil, nil
}

func Converter(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	return []lwm2m.Lwm2mObject{lwm2m.NewDevice(deviceID, ts)}, nil
}
