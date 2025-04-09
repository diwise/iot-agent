package application

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

	"github.com/diwise/iot-agent/internal/pkg/application/decoders"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	core "github.com/diwise/iot-core/pkg/messaging/events"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	dmtypes "github.com/diwise/iot-device-mgmt/pkg/types"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

//go:generate moq -rm -out iotagent_mock.go . App

const UNKNOWN = "unknown"

type App interface {
	HandleSensorEvent(ctx context.Context, se types.SensorEvent) error
	HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error
	GetDevice(ctx context.Context, deviceID string) (dmc.Device, error)
	Save(ctx context.Context, se types.SensorEvent) error
}

type app struct {
	registry decoders.Registry
	client   dmc.DeviceManagementClient
	msgCtx   messaging.MsgContext
	store    storage.Storage

	notFoundDevices            map[string]time.Time
	notFoundDevicesMu          sync.Mutex
	createUnknownDeviceEnabled bool
	createUnknownDeviceTenant  string
}

func New(dmc dmc.DeviceManagementClient, msgCtx messaging.MsgContext, storage storage.Storage, createUnknownDeviceEnabled bool, createUnknownDeviceTenant string) App {
	d := decoders.NewRegistry()

	return &app{
		registry:                   d,
		client:                     dmc,
		msgCtx:                     msgCtx,
		store:                      storage,
		notFoundDevices:            make(map[string]time.Time),
		createUnknownDeviceEnabled: createUnknownDeviceEnabled,
		createUnknownDeviceTenant:  createUnknownDeviceTenant,
	}
}

func (a *app) Save(ctx context.Context, se types.SensorEvent) error {
	//TODO: remove from interface and use internal only
	return a.store.Save(ctx, se)
}

func (a *app) HandleSensorEvent(ctx context.Context, se types.SensorEvent) error {
	devEUI := strings.ToLower(se.DevEui)

	log := logging.GetFromContext(ctx)
	log = log.With(slog.String("devEUI", devEUI))

	ctx = logging.NewContextWithLogger(ctx, log)

	device, err := a.findDevice(ctx, devEUI, a.client.FindDeviceFromDevEUI)
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

	msg := types.StatusMessage{
		DeviceID:     device.ID(),
		BatteryLevel: 0,
		Code:         0,
		Messages:     []string{},
		Tenant:       device.Tenant(),
		Timestamp:    time.Now().UTC(),
	}

	decoder, converter, ok := a.registry.Get(ctx, device.SensorType())
	if !ok {
		log.Debug("no decoder found for device type", "device_type", device.SensorType())
		return nil
	}

	payload, err := decoder(ctx, se)
	if err != nil {
		log.Error("failed to decode message", "err", err.Error())
		return err
	}

	objects, err := converter(ctx, device.ID(), payload, se.Timestamp)
	if err != nil {
		decoderErr, ok := err.(*types.DecoderErr)

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

	types := device.Types()
	if !slices.Contains(types, "urn:oma:lwm2m:ext:3") { // always handle urn:oma:lwm2m:ext:3 = Device
		types = append(types, "urn:oma:lwm2m:ext:3")
	}

	for _, obj := range objects {
		if !slices.Contains(types, obj.ObjectURN()) {
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

func (a *app) createUnknownDevice(ctx context.Context, se types.SensorEvent) error {
	d := dmtypes.Device{
		Active:   false,
		DeviceID: deterministicGUID(se.DevEui),
		SensorID: se.DevEui,
		Name:     se.DeviceName,
		DeviceProfile: dmtypes.DeviceProfile{
			Name:    UNKNOWN,
			Decoder: UNKNOWN,
		},
		Tenant: a.createUnknownDeviceTenant,
	}

	return a.client.CreateDevice(ctx, d)
}

func (a *app) HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error {
	deviceID = strings.ToLower(deviceID)

	log := logging.GetFromContext(ctx).With(slog.String("device_id", deviceID))
	ctx = logging.NewContextWithLogger(ctx, log)

	device, err := a.findDevice(ctx, deviceID, a.client.FindDeviceFromInternalID)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Warn("blacklisted")
			return nil
		}

		return err
	}

	msg := types.StatusMessage{
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
	return a.client.FindDeviceFromInternalID(ctx, deviceID)
}

func (a *app) handleSensorMeasurementList(ctx context.Context, pack senml.Pack) error {
	log := logging.GetFromContext(ctx)
	m := core.NewMessageReceived(pack)

	err := a.msgCtx.SendCommandTo(ctx, m, "iot-core")
	if err != nil {
		log.Error("could not send message.received to iot-core", "err", err.Error())
		return err
	}

	b, _ := json.Marshal(m)
	log.Debug("message received sent to iot-core", "body", string(b))

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

func (a *app) sendStatusMessage(ctx context.Context, msg types.StatusMessage, tenant string) bool {
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
