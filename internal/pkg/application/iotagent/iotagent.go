package iotagent

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	core "github.com/diwise/iot-core/pkg/messaging/events"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/iot-device-mgmt/pkg/types"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

//go:generate moq -rm -out iotagent_mock.go . App

const UNKNOWN = "unknown"

type App interface {
	HandleSensorEvent(ctx context.Context, se application.SensorEvent) error
	HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error
	GetDevice(ctx context.Context, deviceID string) (dmc.Device, error)
}

type app struct {
	decoderRegistry        decoder.DecoderRegistry
	deviceManagementClient dmc.DeviceManagementClient
	msgCtx                 messaging.MsgContext

	notFoundDevices            map[string]time.Time
	notFoundDevicesMu          sync.Mutex
	createUnknownDeviceEnabled bool
	createUnknownDeviceTenant  string
}

func New(dmc dmc.DeviceManagementClient, msgCtx messaging.MsgContext, createUnknownDeviceEnabled bool, createUnknownDeviceTenant string) App {
	d := decoder.NewDecoderRegistry()

	return &app{
		decoderRegistry:            d,
		deviceManagementClient:     dmc,
		msgCtx:                     msgCtx,
		notFoundDevices:            make(map[string]time.Time),
		createUnknownDeviceEnabled: createUnknownDeviceEnabled,
		createUnknownDeviceTenant:  createUnknownDeviceTenant,
	}
}

func (a *app) HandleSensorEvent(ctx context.Context, se application.SensorEvent) error {
	devEUI := strings.ToLower(se.DevEui)

	log := logging.GetFromContext(ctx)
	log = log.With(slog.String("devEUI", devEUI))

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
		a.ignoreDeviceFor(devEUI, 1*time.Hour)
		return nil
	}

	log = log.With(slog.String("device_id", device.ID()), slog.String("type", device.SensorType()))
	ctx = logging.NewContextWithLogger(ctx, log)

	msg := StatusMessage{
		DeviceID:     device.ID(),
		BatteryLevel: 0,
		Code:         0,
		Messages:     []string{},
		Tenant:       device.Tenant(),
		Timestamp:    time.Now().UTC(),
	}

	decoder := a.decoderRegistry.GetDecoderForSensorType(ctx, device.SensorType())
	objects, err := decoder(ctx, device.ID(), se)
	if err != nil {
		decoderErr, ok := err.(*application.DecoderErr)

		if !ok {
			log.Error("failed to decode message", "err", err.Error())
			return err
		}

		msg.Code = decoderErr.Code
		msg.Messages = decoderErr.Messages
		msg.Timestamp = decoderErr.Timestamp
	}

	if !a.sendStatusMessage(ctx, msg, device.Tenant()) {
		log.Debug("no status message sent for sensor event")
	}

	if !device.IsActive() {
		log.Debug("ignored message from inactive device")
		return nil
	}

	var errs []error
	for _, obj := range objects {
		if !slices.Contains(device.Types(), obj.ObjectURN()) {
			log.Debug("skip object since device should not handle object type", slog.String("object_type", obj.ObjectURN()))
			continue
		}

		err := a.handleSensorMeasurementList(ctx, lwm2m.ToPack(obj))
		if err != nil {
			log.Error("could not handle measurement", "err", err.Error())
			errs = append(errs, err)
			continue
		}
	}

	return errors.Join(errs...)
}

func (a *app) isDeviceUnknown(device dmc.Device) bool {
	return device.SensorType() == UNKNOWN
}

func deterministicGUID(input string) string {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	hash := hasher.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], hash[:16])

	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func (a *app) createUnknownDevice(ctx context.Context, se application.SensorEvent) error {
	d := types.Device{
		Active:   false,
		DeviceID: deterministicGUID(se.DevEui),
		SensorID: se.DevEui,
		Name:     se.DeviceName,
		DeviceProfile: types.DeviceProfile{
			Name: UNKNOWN,
		},
		Tenant: a.createUnknownDeviceTenant,
	}

	return a.deviceManagementClient.CreateDevice(ctx, d)
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

	msg := StatusMessage{
		DeviceID:     device.ID(),
		BatteryLevel: 0,
		Code:         0,
		Messages:     []string{},
		Tenant:       device.Tenant(),
		Timestamp:    time.Now().UTC(),
	}

	if !a.sendStatusMessage(ctx, msg, device.Tenant()) {
		log.Debug("no status message sent for measurement list")
	}

	return a.handleSensorMeasurementList(ctx, pack)
}

func (a *app) GetDevice(ctx context.Context, deviceID string) (dmc.Device, error) {
	return a.deviceManagementClient.FindDeviceFromInternalID(ctx, deviceID)
}

func (a *app) handleSensorMeasurementList(ctx context.Context, pack senml.Pack) error {
	log := logging.GetFromContext(ctx)
	m := core.NewMessageReceived(pack)

	err := a.msgCtx.SendCommandTo(ctx, &m, "iot-core")
	if err != nil {
		log.Error("could not send message.received to iot-core", "err", err.Error())
		return err
	}

	return nil
}

var errDeviceOnBlackList = errors.New("blacklisted")

func (a *app) deviceIsCurrentlyIgnored(_ context.Context, id string) bool {
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
		a.ignoreDeviceFor(id, 1*time.Hour)
		return nil, fmt.Errorf("device lookup failure (%w)", err)
	}

	return device, nil
}

func (a *app) ignoreDeviceFor(id string, period time.Duration) {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	a.notFoundDevices[id] = time.Now().UTC().Add(period)
}

func (a *app) sendStatusMessage(ctx context.Context, msg StatusMessage, tenant string) bool {
	log := logging.GetFromContext(ctx)

	if msg.Tenant == "" {
		log.Warn("tenant information is missing")
		msg.Tenant = tenant
	}

	if msg.DeviceID == "" {
		log.Debug("deviceID is missing from status message")
		return false
	}

	err := a.msgCtx.PublishOnTopic(ctx, &msg)
	if err != nil {
		log.Error("failed to publish status message", "err", err.Error())
		return false
	}

	log.Debug("status message sent")
	return true
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
