package decoder

import (
	"context"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type MessageDecoderFunc func(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error)

func DefaultDecoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	log := logging.GetFromContext(ctx)
	log.Info(fmt.Sprintf("default decoder used for deviceid %s of type %s", deviceID, e.SensorType))

	d := lwm2m.Device{
		ID_:        deviceID,
		Timestamp_: e.Timestamp,
	}

	return []lwm2m.Lwm2mObject{d}, nil
}
