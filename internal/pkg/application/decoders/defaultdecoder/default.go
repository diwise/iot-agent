package defaultdecoder

import (
	"context"
	"log/slog"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

func DefaultDecoder(ctx context.Context, deviceID string, e types.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	log := logging.GetFromContext(ctx)
	log.Info("default decoder used", slog.String("sensor_type", e.SensorType))

	return []lwm2m.Lwm2mObject{lwm2m.NewDevice(deviceID, e.Timestamp)}, nil
}
