package iotagent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/messageprocessor"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

//go:generate moq -rm -out iotagent_mock.go . App

type App interface {
	HandleSensorEvent(ctx context.Context, se application.SensorEvent) error
}

type app struct {
	messageProcessor       messageprocessor.MessageProcessor
	decoderRegistry        decoder.DecoderRegistry
	deviceManagementClient dmc.DeviceManagementClient
	notFoundDevices        map[string]time.Time
	notFoundDevicesMu      sync.Mutex
}

func New(dmc dmc.DeviceManagementClient, eventPub events.EventSender) App {
	c := conversion.NewConverterRegistry()
	d := decoder.NewDecoderRegistry()
	m := messageprocessor.NewMessageReceivedProcessor(c, eventPub)

	return &app{
		messageProcessor:       m,
		decoderRegistry:        d,
		deviceManagementClient: dmc,
		notFoundDevices:        make(map[string]time.Time),
	}
}

func (a *app) HandleSensorEvent(ctx context.Context, se application.SensorEvent) error {
	log := logging.GetFromContext(ctx).With().Str("devEui", se.DevEui).Logger()
	ctx = logging.NewContextWithLogger(ctx, log)

	device, err := a.findDevice(ctx, se.DevEui)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Warn().Str("deviceName", se.DeviceName).Msg("blacklisted")
			return nil
		}

		return err
	}

	log = log.With().Str("device", device.ID()).Logger()
	ctx = logging.NewContextWithLogger(ctx, log)

	log.Debug().Str("type", device.SensorType()).Msg("message received")

	decoderFn := decoder.PayloadErrorDecoder
	if !se.HasError() {
		decoderFn = a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())
	}

	err = decoderFn(ctx, se, func(ctx context.Context, p payload.Payload) error {
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

var errDeviceOnBlackList = errors.New("blacklisted")

func (a *app) findDevice(ctx context.Context, devEui string) (dmc.Device, error) {

	if a.deviceIsCurrentlyIgnored(ctx, devEui) {
		return nil, errDeviceOnBlackList
	}

	device, err := a.deviceManagementClient.FindDeviceFromDevEUI(ctx, devEui)
	if err != nil {
		a.ignoreDeviceFor(ctx, devEui, 1*time.Hour)
		return nil, fmt.Errorf("device lookup failure (%w)", err)
	}

	return device, nil
}

func (a *app) deviceIsCurrentlyIgnored(ctx context.Context, devEui string) bool {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	if timeOfNextAllowedRetry, ok := a.notFoundDevices[devEui]; ok {
		if !time.Now().UTC().After(timeOfNextAllowedRetry) {
			return true
		}

		delete(a.notFoundDevices, devEui)
	}

	return false
}

func (a *app) ignoreDeviceFor(ctx context.Context, devEui string, period time.Duration) {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	a.notFoundDevices[devEui] = time.Now().UTC().Add(period)
}
