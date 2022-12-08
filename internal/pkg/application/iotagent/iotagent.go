package iotagent

import (
	"context"
	"fmt"
	"time"

	app "github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/messageprocessor"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

//go:generate moq -rm -out iotagent_mock.go . IoTAgent

type IoTAgent interface {
	MessageReceived(ctx context.Context, ue app.SensorEvent) error
	MessageReceivedFn(ctx context.Context, msg []byte, ue app.UplinkASFunc) error
}

type iotAgent struct {
	messageProcessor       messageprocessor.MessageProcessor
	decoderRegistry        decoder.DecoderRegistry
	deviceManagementClient dmc.DeviceManagementClient
	notFoundDevices        map[string]time.Time
}

func NewIoTAgent(dmc dmc.DeviceManagementClient, eventPub events.EventSender) IoTAgent {
	c := conversion.NewConverterRegistry()
	d := decoder.NewDecoderRegistry()
	m := messageprocessor.NewMessageReceivedProcessor(c, eventPub)

	return &iotAgent{
		messageProcessor:       m,
		decoderRegistry:        d,
		deviceManagementClient: dmc,
		notFoundDevices:        make(map[string]time.Time),
	}
}

func (a *iotAgent) MessageReceivedFn(ctx context.Context, msg []byte, ueFunc app.UplinkASFunc) error {
	ue, err := ueFunc(msg)
	if err != nil {
		return err
	}
	return a.MessageReceived(ctx, ue)
}

func (a *iotAgent) MessageReceived(ctx context.Context, ue app.SensorEvent) error {
	log := logging.GetFromContext(ctx).With().Str("devEui", ue.DevEui).Logger()
	ctx = logging.NewContextWithLogger(ctx, log)

	if timeForFirstError, ok := a.notFoundDevices[ue.DevEui]; ok {
		if time.Now().UTC().After(timeForFirstError.Add(1 * time.Hour)) {
			delete(a.notFoundDevices, ue.DevEui)
		} else {
			log.Info().Msg("blacklisted")
			return nil
		}
	}

	device, err := a.deviceManagementClient.FindDeviceFromDevEUI(ctx, ue.DevEui)
	if err != nil {
		a.notFoundDevices[ue.DevEui] = time.Now().UTC()
		return fmt.Errorf("device lookup failure (%w)", err)
	}

	log.Debug().Str("type", device.SensorType()).Msg("message received")

	var decoderFn decoder.MessageDecoderFunc
	if ue.HasError() {
		decoderFn = decoder.PayloadErrorDecoder
	} else {
		decoderFn = a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())
	}

	err = decoderFn(ctx, ue, func(ctx context.Context, p payload.Payload) error {
		err := a.messageProcessor.ProcessMessage(ctx, p, device)
		if err != nil {
			err = fmt.Errorf("failed to process message (%w)", err)
		}
		return err
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to handle received message")
	}

	return err
}
