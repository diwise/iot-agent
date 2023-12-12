package decoder

import (
	"context"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type MessageDecoderFunc func(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error)

func DefaultDecoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	log := logging.GetFromContext(ctx)
	log.Info(fmt.Sprintf("default decoder used for deviceid %s of type %s", deviceID, e.SensorType))

	return []lwm2m.Lwm2mObject{lwm2m.NewDevice(deviceID, e.Timestamp)}, nil
}
