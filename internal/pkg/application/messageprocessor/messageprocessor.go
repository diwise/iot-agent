package messageprocessor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	core "github.com/diwise/iot-core/pkg/messaging/events"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/farshidtz/senml/v2"
)

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, p payload.Payload, device dmc.Device) error
}

type msgProcessor struct {
	conReg conversion.ConverterRegistry
	event  events.EventSender
}

func NewMessageReceivedProcessor(conReg conversion.ConverterRegistry, event events.EventSender) MessageProcessor {
	return &msgProcessor{
		conReg: conReg,
		event:  event,
	}
}

func (mp *msgProcessor) ProcessMessage(ctx context.Context, p payload.Payload, device dmc.Device) error {
	log := logging.GetFromContext(ctx)

	var d []func(*events.StatusMessage)
	d = append(d, events.WithStatus(p.Status().Code, p.Status().Messages))
	if bat, ok := payload.Get[int](p, "batteryLevel"); ok {
		d = append(d, events.WithBatteryLevel(bat))
	}

	err := mp.event.Publish(ctx, events.NewStatusMessage(device.ID(), d...))
	if err != nil {
		log.Error().Err(err).Msg("failed to publish status message")
	}

	if p.Status().Code == payload.PayloadError {
		log.Info().Msg("ignoring payload due to device error")
		return nil
	}

	messageConverters := mp.conReg.DesignateConverters(ctx, device.Types())
	if len(messageConverters) == 0 {
		return errors.New("no matching converters for device")
	}

	conversionErrors := make([]error, 0, len(messageConverters))

	for _, convert := range messageConverters {
		err := convert(ctx, device.ID(), p, func(sp senml.Pack) error {
			if err := sp.Validate(); err != nil {
				return fmt.Errorf("invalid senML package: %w", err)
			}

			m := core.MessageReceived{
				Device:    device.ID(),
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Pack:      sp,
			}

			if device.IsActive() {
				if err := mp.event.Send(ctx, &m); err != nil {
					return fmt.Errorf("failed to send event: %w", err)
				}

				log.Debug().Msg("event published")
			}

			return nil
		})

		if err != nil {
			conversionErrors = append(conversionErrors, err)
		}
	}

	if len(conversionErrors) > 0 {
		log.Warn().Msgf("%d out of %d converters failed: %v", len(conversionErrors), len(messageConverters), conversionErrors)
	}

	if !device.IsActive() {
		log.Warn().Msg("ignored message from inactive device")
	}

	return nil
}
