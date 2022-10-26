package messageprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	iotcore "github.com/diwise/iot-core/pkg/messaging/events"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, p payload.Payload) error
}

type msgProcessor struct {
	dmc    dmc.DeviceManagementClient
	conReg conversion.ConverterRegistry
	event  events.EventSender
}

func NewMessageReceivedProcessor(dmc dmc.DeviceManagementClient, conReg conversion.ConverterRegistry, event events.EventSender) MessageProcessor {
	return &msgProcessor{
		dmc:    dmc,
		conReg: conReg,
		event:  event,
	}
}

func (mp *msgProcessor) ProcessMessage(ctx context.Context, p payload.Payload) error {
	log := logging.GetFromContext(ctx)

	device, err := mp.dmc.FindDeviceFromDevEUI(ctx, p.DevEui())
	if err != nil {
		log.Error().Err(err).Msg("device lookup failure")
		return err
	}

	var d []func(*events.StatusMessage)
	d = append(d, events.WithStatus(p.Status().Code, p.Status().Messages))
	if bat, ok := payload.Get[int](p, "batteryLevel"); ok {
		d = append(d, events.WithBatteryLevel(bat))
	}

	err = mp.event.Publish(ctx, events.NewStatusMessage(device.ID(), d...))
	if err != nil {
		log.Error().Err(err).Msg("failed to publish status message")
	}

	if p.Status().Code == payload.PayloadError {
		log.Info().Msg("ignoring payload due to device error")
		return nil
	}

	messageConverters := mp.conReg.DesignateConverters(ctx, device.Types())
	if len(messageConverters) == 0 {
		return fmt.Errorf("no matching converters for device")
	}

	for _, convert := range messageConverters {
		pack, err := convert(ctx, device.ID(), p)
		if err != nil {
			log.Error().Err(err).Msg("conversion failed")
			continue
		}

		m := iotcore.MessageReceived{
			Device:    device.ID(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Pack:      pack,
		}

		if device.IsActive() {
			err = mp.event.Send(ctx, &m)
			if err != nil {
				log.Error().Err(err).Msg("failed to send event")
			}
		}
	}

	if !device.IsActive() {
		log.Warn().Str("deviceID", device.ID()).Msg("ignoring message from inactive device")
	}

	return nil
}
