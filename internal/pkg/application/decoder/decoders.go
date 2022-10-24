package decoder

import (
	"context"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type MessageDecoderFunc func(context.Context, application.SensorEvent, func(context.Context, payload.Payload) error) error

func DefaultDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	log := logging.GetFromContext(ctx)

	p, err := payload.New(ue.DevEui, ue.Timestamp)
	if err != nil {
		return err
	}

	log.Info().Msgf("default decoder used for devEUI %s", ue.DevEui)

	return fn(ctx, p)
}
