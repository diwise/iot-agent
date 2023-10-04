package decoder

import (
	"context"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type MessageDecoderFunc func(context.Context, application.SensorEvent, func(context.Context, payload.Payload) error) error

func PayloadErrorDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	p, err := payload.New(ue.DevEui, ue.Timestamp, payload.Status(uint8(payload.PayloadError), []string{ue.Error.Type, ue.Error.Message}))
	if err != nil {
		return err
	}
	return fn(ctx, p)
}

func DefaultDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	log := logging.GetFromContext(ctx)

	p, err := payload.New(ue.DevEui, ue.Timestamp)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("default decoder used for devEUI %s of type %s", ue.DevEui, ue.SensorType))

	return fn(ctx, p)
}
