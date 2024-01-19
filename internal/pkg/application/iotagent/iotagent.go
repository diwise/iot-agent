package iotagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	core "github.com/diwise/iot-core/pkg/messaging/events"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/iot-device-mgmt/pkg/types"
	"github.com/diwise/messaging-golang/pkg/messaging"
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
	decoderRegistry        decoder.DecoderRegistry
	deviceManagementClient dmc.DeviceManagementClient
	msgCtx                 messaging.MsgContext
	storage                storage.Storage

	notFoundDevices            map[string]time.Time
	notFoundDevicesMu          sync.Mutex
	createUnknownDeviceEnabled bool
	createUnknownDeviceTenant  string

}

func New(dmc dmc.DeviceManagementClient, msgCtx messaging.MsgContext, store storage.Storage, createUnknownDeviceEnabled bool, createUnknownDeviceTenant string) App {
	d := decoder.NewDecoderRegistry()

	return &app{
		decoderRegistry:        d,
		deviceManagementClient: dmc,
		msgCtx:                 msgCtx,
		storage:                store,
		notFoundDevices:        make(map[string]time.Time),
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
			err := a.createUnknownDevice(ctx, se)
			if err != nil {
				log.Error("could not create unknown device", "err", err.Error())				
			}
		}

		return err
	}

	if a.createUnknownDeviceEnabled && a.isDeviceUnknown(device) {
		a.ignoreDeviceFor(ctx, devEUI, 1*time.Hour)
		return nil
	}

	log = log.With(slog.String("device_id", device.ID()), slog.String("type", device.SensorType()))
	ctx = logging.NewContextWithLogger(ctx, log)

	decoder := a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())
	objects, err := decoder(ctx, device.ID(), se)
	if err != nil {
		log.Error("failed to decode message", "err", err.Error())
		return err
	}

	a.sendStatusMessage(ctx, device.ID(), device.Tenant(), nil)

	err = a.storage.AddMany(ctx, device.ID(), lwm2m.ToPacks(objects), time.Now().UTC())
	if err != nil {
		log.Error("could not store measurements", "err", err.Error())
		return err
	}

	if device.IsActive() {
		var errs []error
		for _, obj := range objects {
			err := a.handleSensorMeasurementList(ctx, device.ID(), lwm2m.ToPack(obj))
			if err != nil {
				log.Error("could not handle measurement", "err", err.Error())
				errs = append(errs, err)
				// TODO: handle error
			}
		}
		return errors.Join(errs...)
	} else {
		log.Warn("ignored message from inactive device")
	}

	return nil
}

func (a *app) isDeviceUnknown(device dmc.Device) bool {
	return device.SensorType() == UNKNOWN
}

func (a *app) createUnknownDevice(ctx context.Context, se application.SensorEvent) error {
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

	return  a.deviceManagementClient.CreateDevice(ctx, d)
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

	a.msgCtx.SendCommandTo(ctx, &m, "iot-core")

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

func (a *app) sendStatusMessage(ctx context.Context, deviceID, tenant string, l lwm2m.Lwm2mObject) {
	log := logging.GetFromContext(ctx)

	msg := &StatusMessage{
		DeviceID:  deviceID,
		Tenant:    tenant,
		Timestamp: time.Now().UTC(),
	}

	// TODO: status codes and messages?

	if msg.Tenant == "" {
		log.Warn("tenant information is missing")
	}

	err := a.msgCtx.PublishOnTopic(ctx, msg)
	if err != nil {
		log.Error("failed to publish status message", "err", err.Error())
	}
}

type StatusMessage struct {
	DeviceID     string    `json:"deviceID"`
	BatteryLevel int       `json:"batteryLevel"`
	Code         int       `json:"statusCode"`
	Messages     []string  `json:"statusMessages,omitempty"`
	Tenant       string    `json:"tenant"`
	Timestamp    time.Time `json:"timestamp"`
}

func (m *StatusMessage) ContentType() string {
	return "application/json"
}

func (m *StatusMessage) TopicName() string {
	return "device-status"
}

func (m *StatusMessage) Body() []byte {
	b, _ := json.Marshal(m)
	return b
}
