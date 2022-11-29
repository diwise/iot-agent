package iotagent

import (
	"context"
	"fmt"

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
}

func NewIoTAgent(dmc dmc.DeviceManagementClient, eventPub events.EventSender) IoTAgent {
	conreg := conversion.NewConverterRegistry()
	decreg := decoder.NewDecoderRegistry()
	msgprcs := messageprocessor.NewMessageReceivedProcessor(conreg, eventPub)

	return &iotAgent{
		messageProcessor:       msgprcs,
		decoderRegistry:        decreg,
		deviceManagementClient: dmc,
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
	device, err := a.deviceManagementClient.FindDeviceFromDevEUI(ctx, ue.DevEui)
	if err != nil {
		return fmt.Errorf("device lookup failure (%w)", err)
	}

	log := logging.GetFromContext(ctx)
	log.Debug().Msgf("MessageReceived with device %s of type %s", device.ID(), device.SensorType())

	decoderFn := a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())

	err = decoderFn(ctx, ue, func(ctx context.Context, p payload.Payload) error {
		err := a.messageProcessor.ProcessMessage(ctx, p, device)
		if err != nil {
			err = fmt.Errorf("failed to process message (%w)", err)
		}
		return err
	})

	return err
}
