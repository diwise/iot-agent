package iotagent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/messageprocessor"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	core "github.com/diwise/iot-core/pkg/messaging/events"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/iot-device-mgmt/pkg/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/farshidtz/senml/v2"
	"github.com/google/uuid"
)

//go:generate moq -rm -out iotagent_mock.go . App

const UNKNOWN = "unknown"

type App interface {
	HandleSensorEvent(ctx context.Context, se application.SensorEvent) error
	HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error
	GetMeasurements(ctx context.Context, deviceID string, temprel string, t, et time.Time, lastN int) ([]application.Measurement, error)
	GetDevice(ctx context.Context, deviceID string) (dmc.Device, error)
}

type app struct {
	messageProcessor       messageprocessor.MessageProcessor
	decoderRegistry        decoder.DecoderRegistry
	deviceManagementClient dmc.DeviceManagementClient
	eventSender            events.EventSender
	storage                storage.Storage

	notFoundDevices            map[string]time.Time
	notFoundDevicesMu          sync.Mutex
	createUnknownDeviceEnabled bool
	createUnknownDeviceTenant  string
}

func New(dmc dmc.DeviceManagementClient, eventPub events.EventSender, store storage.Storage, createUnknownDeviceEnabled bool, createUnknownDeviceTenant string) App {
	c := conversion.NewConverterRegistry()
	d := decoder.NewDecoderRegistry()
	m := messageprocessor.NewMessageReceivedProcessor(c)

	return &app{
		messageProcessor:           m,
		decoderRegistry:            d,
		deviceManagementClient:     dmc,
		eventSender:                eventPub,
		storage:                    store,
		notFoundDevices:            make(map[string]time.Time),
		createUnknownDeviceEnabled: createUnknownDeviceEnabled,
		createUnknownDeviceTenant:  createUnknownDeviceTenant,
	}
}

func (a *app) HandleSensorEvent(ctx context.Context, se application.SensorEvent) error {
	devEUI := strings.ToLower(se.DevEui)
	log := logging.GetFromContext(ctx).With(slog.String("devEui", devEUI))
	ctx = logging.NewContextWithLogger(ctx, log)

	device, err := a.findDevice(ctx, devEUI, a.deviceManagementClient.FindDeviceFromDevEUI)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Warn("blacklisted", "deviceName", se.DeviceName)
			return nil
		}

		if a.createUnknownDeviceEnabled {
			a.createUnknownDevice(ctx, se)
		}
		return err
	}

	if a.createUnknownDeviceEnabled && a.isDeviceUnknown(device) {
		a.ignoreDeviceFor(ctx, devEUI, 1*time.Hour)
		return nil
	}

	log = log.With(
		slog.String("device_id", device.ID()),
		slog.String("type", device.SensorType()),
	)
	ctx = logging.NewContextWithLogger(ctx, log)

	log.Debug("message received")

	decodePayload := decoder.PayloadErrorDecoder
	if !se.HasError() {
		decodePayload = a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())
	}

	err = decodePayload(ctx, se, func(ctx context.Context, p payload.Payload) error {
		a.sendStatusMessage(ctx, device.ID(), device.Tenant(), p)

		packs, err := a.messageProcessor.ProcessMessage(ctx, p, device)
		if err != nil {
			return fmt.Errorf("failed to process message (%w)", err)
		}

		err = a.storage.AddMany(ctx, device.ID(), packs, time.Now().UTC())
		if err != nil {
			log.Error("could not store measurements", "err", err.Error())
			return err
		}

		if device.IsActive() {
			for _, pack := range packs {
				a.handleSensorMeasurementList(ctx, device.ID(), pack)
			}
		} else {
			log.Warn("ignored message from inactive device")
		}

		return nil
	})

	if err != nil {
		log.Error("failed to handle received message", "err", err.Error())
	}

	return err
}

func (a *app) isDeviceUnknown(device dmc.Device) bool {
	return device.SensorType() == UNKNOWN
}

func (a *app) createUnknownDevice(ctx context.Context, se application.SensorEvent) {
	logger := logging.GetFromContext(ctx).With(slog.String("func", "createUnknownDevice"))

	d := types.Device{
		Active:   false,
		DeviceID: uuid.New().String(),
		SensorID: se.DevEui,
		Name:     se.DeviceName,
		DeviceProfile: types.DeviceProfile{
			Name: UNKNOWN,
		},
		Tenant: types.Tenant{
			Name: a.createUnknownDeviceTenant,
		},
	}

	err := a.deviceManagementClient.CreateDevice(ctx, d)
	if err != nil {
		logger.Error("failed to create unknown device", "err", err.Error())
	}
}

func (a *app) HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error {
	deviceID = strings.ToLower(deviceID)
	log := logging.GetFromContext(ctx).With(slog.String("device_id", deviceID))
	ctx = logging.NewContextWithLogger(ctx, log)

	device, err := a.findDevice(ctx, deviceID, a.deviceManagementClient.FindDeviceFromInternalID)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Warn("blacklisted")
			return nil
		}

		return err
	}

	err = a.storage.Add(ctx, device.ID(), pack, time.Now().UTC())
	if err != nil {
		log.Error("could not store measurement", "err", err.Error())
		return err
	}

	a.sendStatusMessage(ctx, device.ID(), device.Tenant(), nil)

	return a.handleSensorMeasurementList(ctx, device.ID(), pack)
}

func (a *app) GetDevice(ctx context.Context, deviceID string) (dmc.Device, error) {
	return a.deviceManagementClient.FindDeviceFromInternalID(ctx, deviceID)
}

func (a *app) GetMeasurements(ctx context.Context, deviceID string, temprel string, t, et time.Time, lastN int) ([]application.Measurement, error) {
	if temprel == "before" {
		et = t
		t = time.Unix(0, 0)
	}

	if temprel == "after" {
		et = time.Now().UTC()
	}

	rows, err := a.storage.GetMeasurements(ctx, deviceID, temprel, t, et, lastN)
	if err != nil {
		return []application.Measurement{}, err
	}

	measurements := make([]application.Measurement, len(rows))

	for i, m := range rows {
		measurements[i] = application.Measurement{
			Timestamp: m.Timestamp,
			Pack:      m.Pack,
		}
	}

	return measurements, nil
}

func (a *app) handleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error {
	m := core.MessageReceived{
		Device:    deviceID,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Pack:      pack,
	}

	a.eventSender.Send(ctx, &m)

	return nil
}

var errDeviceOnBlackList = errors.New("blacklisted")

func (a *app) deviceIsCurrentlyIgnored(ctx context.Context, id string) bool {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	if timeOfNextAllowedRetry, ok := a.notFoundDevices[id]; ok {
		if !time.Now().UTC().After(timeOfNextAllowedRetry) {
			return true
		}

		delete(a.notFoundDevices, id)
	}

	return false
}

func (a *app) findDevice(ctx context.Context, id string, finder func(ctx context.Context, id string) (dmc.Device, error)) (dmc.Device, error) {
	if a.deviceIsCurrentlyIgnored(ctx, id) {
		return nil, errDeviceOnBlackList
	}

	device, err := finder(ctx, id)
	if err != nil {
		a.ignoreDeviceFor(ctx, id, 1*time.Hour)
		return nil, fmt.Errorf("device lookup failure (%w)", err)
	}

	return device, nil
}

func (a *app) ignoreDeviceFor(ctx context.Context, id string, period time.Duration) {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	a.notFoundDevices[id] = time.Now().UTC().Add(period)
}

func (a *app) sendStatusMessage(ctx context.Context, deviceID, tenant string, p payload.Payload) {
	logger := logging.GetFromContext(ctx).With(slog.String("func", "sendStatusMessage"))

	var decorators []func(*events.StatusMessage)

	decorators = append(decorators, events.WithTenant(tenant))

	if p != nil {
		decorators = append(decorators, events.WithStatus(p.Status().Code, p.Status().Messages))
	} else {
		decorators = append(decorators, events.WithStatus(0, []string{}))
	}

	if bat, ok := payload.Get[int](p, payload.BatteryLevelProperty); ok {
		decorators = append(decorators, events.WithBatteryLevel(bat))
	}

	msg := events.NewStatusMessage(deviceID, decorators...)

	if msg.Tenant == "" {
		logger.Warn("tenant information is missing")
	}

	err := a.eventSender.Publish(ctx, msg)
	if err != nil {
		logger.Error("failed to publish status message", "err", err.Error())
	}
}
