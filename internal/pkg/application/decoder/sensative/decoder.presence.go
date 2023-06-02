package sensative

import (
	"context"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func PresenceDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	return Decoder(ctx, ue, fn)
}
