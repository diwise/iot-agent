package iotagent

import (
	"context"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
)

//go:generate moq -rm -out iotagent_mock.go . IoTAgent

type IoTAgent interface {
	MessageReceived(ctx context.Context, ue mqtt.UplinkEvent) error
	MessageReceivedFn(ctx context.Context, msg []byte, ue mqtt.UplinkASFunc) error
}

type iotAgent struct {
	messageProcessor       messageprocessor.MessageProcessor
	decoderRegistry        decoder.DecoderRegistry
	deviceManagementClient dmc.DeviceManagementClient
}

func NewIoTAgent(dmc dmc.DeviceManagementClient, eventPub events.EventSender) IoTAgent {
	conreg := conversion.NewConverterRegistry()
	decreg := decoder.NewDecoderRegistry()
	msgprcs := messageprocessor.NewMessageReceivedProcessor(dmc, conreg, eventPub)

	return &iotAgent{
		messageProcessor:       msgprcs,
		decoderRegistry:        decreg,
		deviceManagementClient: dmc,
	}
}

func (a *iotAgent) MessageReceivedFn(ctx context.Context, msg []byte, ueFunc mqtt.UplinkASFunc) error {
	ue, err := ueFunc(msg)
	if err != nil {
		return err
	}
	return a.MessageReceived(ctx, ue)
}

func (a *iotAgent) MessageReceived(ctx context.Context, ue mqtt.UplinkEvent) error {
	device, err := a.deviceManagementClient.FindDeviceFromDevEUI(ctx, ue.DevEui)
	if err != nil {
		return fmt.Errorf("device lookup failure (%w)", err)
	}

	decoderFn := a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())

	err = decoderFn(ctx, ue, func(ctx context.Context, payload decoder.Payload) error {
		err := a.messageProcessor.ProcessMessage(ctx, payload)
		if err != nil {
			err = fmt.Errorf("failed to process message (%w)", err)
		}
		return err
	})

	return err
}
