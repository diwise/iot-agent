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
	"github.com/diwise/iot-device-mgmt/pkg/client"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	dmtypes "github.com/diwise/iot-device-mgmt/pkg/types"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

var errDeviceOnBlackList = errors.New("blacklisted")
var errDeviceNotFound = errors.New("device not found")
var errDeviceIgnored = errors.New("device is ignored")
var errEventContainsNoPayload = errors.New("event contains no payload")

//go:generate moq -rm -out iotagent_mock.go . App

const UNKNOWN = "unknown"

type App interface {
	HandleSensorEvent(ctx context.Context, se types.Event) error
	HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error
	GetDevice(ctx context.Context, deviceID string) (dmc.Device, error)
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

func (a *app) GetDevice(ctx context.Context, deviceID string) (dmc.Device, error) {
	return a.client.FindDeviceFromInternalID(ctx, deviceID)
}

func (a *app) decodeAndConvert(ctx context.Context, se types.Event) (dmc.Device, types.SensorPayload, []lwm2m.Lwm2mObject, error) {
	device, err := a.findDevice(ctx, se.DevEUI, a.client.FindDeviceFromDevEUI)
	if err != nil {
		return nil, nil, nil, err
	}

	if a.createUnknownDeviceEnabled && device.SensorType() == UNKNOWN {
		a.ignoreDeviceFor(se.DevEUI, 1*time.Hour)
		return device, nil, nil, errDeviceIgnored
	}

	if se.Payload == nil {
		return device, nil, nil, errEventContainsNoPayload
	}

	decoder, converter, ok := a.registry.Get(ctx, device.SensorType())
	if !ok {
		return device, nil, nil, nil
	}

	payload, err := decoder(ctx, se)
	if err != nil {
		return device, nil, nil, err
	}

	objects, err := converter(ctx, device.ID(), payload, se.Timestamp)
	if err != nil {
		return device, nil, nil, err
	}

	return device, payload, objects, nil
}

func (a *app) storeSensorEvent(ctx context.Context, se types.Event, device dmc.Device, payload types.SensorPayload, objects []lwm2m.Lwm2mObject, err error) {
	a.store.Save(ctx, se, device, payload, objects, err)
}

func (a *app) HandleSensorEvent(ctx context.Context, se types.Event) error {
	var errs []error

	log := logging.GetFromContext(ctx).With(slog.String("sensor_id", se.DevEUI))

	var device dmc.Device
	var payload types.SensorPayload
	var objects []lwm2m.Lwm2mObject
	var err error

	device, payload, objects, err = a.decodeAndConvert(ctx, se)
	a.storeSensorEvent(ctx, se, device, payload, objects, err)

	if err != nil {
		if errors.Is(err, errEventContainsNoPayload) {
			log.Debug("event contains no payload")
			return nil
		}

		if errors.Is(err, errDeviceOnBlackList) {
			log.Info("blacklisted")
			return nil
		}

		if errors.Is(err, errDeviceIgnored) {
			log.Debug("device is ignored")
			return nil
		}

		if errors.Is(err, errDeviceNotFound) {
			if a.createUnknownDeviceEnabled {
				err := a.createUnknownDevice(ctx, se)
				if err != nil {
					log.Error("could not create new device of unknown type", "err", err.Error())
					return err
				}

				a.ignoreDeviceFor(se.DevEUI, 1*time.Hour)

				return nil
			}

			log.Debug("device not found")

			return nil
		}

		return err
	}

	err = a.sendStatusMessage(ctx, device, &se, payload)
	if err != nil {
		log.Warn("failed to send status message", "err", err.Error())
	}

	if !device.IsActive() {
		return nil
	}

	types := device.Types()

	for _, obj := range objects {
		if !slices.Contains(types, obj.ObjectURN()) {
			log.Debug(fmt.Sprintf("%s is not in device types list %s", obj.ObjectURN(), strings.Join(types, ", ")))
			continue
		}

		pack := lwm2m.ToPack(obj)

		err := a.handleSensorMeasurementList(ctx, pack)
		if err != nil {
			b, _ := json.Marshal(pack)
			log.Error("could not handle measurement", "pack", string(b), "err", err.Error())
			errs = append(errs, err)
			continue
		}
	}

	log.Debug("sensor measurements processed", "device_id", device.ID())

	return errors.Join(errs...)
}

func (a *app) HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error {
	deviceID = strings.ToLower(deviceID)

	log := logging.GetFromContext(ctx).With(slog.String("device_id", deviceID))
	ctx = logging.NewContextWithLogger(ctx, log)

	d, err := a.findDevice(ctx, deviceID, a.client.FindDeviceFromInternalID)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Info("blacklisted", "device_id", deviceID)
			return nil
		}

		return err
	}

	err = a.sendStatusMessage(ctx, d, nil, nil)
	if err != nil {
		log.Warn("failed to send status message", "err", err.Error())
	}

	err = a.handleSensorMeasurementList(ctx, pack)
	if err != nil {
		log.Error("could not handle measurement list", "err", err.Error())
		return err
	}

	return nil
}

func (a *app) handleSensorMeasurementList(ctx context.Context, pack senml.Pack) error {
	log := logging.GetFromContext(ctx)
	m := core.NewMessageReceived(pack)

	err := a.msgCtx.SendCommandTo(ctx, m, "iot-core")
	if err != nil {
		log.Error("failed to send message.received to iot-core", "err", err.Error())
		return err
	}

	log.Debug("message.received => iot-core", "device_id", m.DeviceID(), "object_id", m.ObjectID(), "tenant", m.Tenant())

	return nil
}

func (a *app) findDevice(ctx context.Context, id string, finder func(ctx context.Context, id string) (dmc.Device, error)) (dmc.Device, error) {
	if a.deviceIsCurrentlyIgnored(ctx, id) {
		return nil, errDeviceOnBlackList
	}

	device, err := finder(ctx, id)
	if err != nil {
		a.ignoreDeviceFor(id, 1*time.Hour)
		return nil, errDeviceNotFound
	}

	return device, nil
}

func (a *app) ignoreDeviceFor(id string, period time.Duration) {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	a.notFoundDevices[id] = time.Now().UTC().Add(period)
}

func (a *app) deviceIsCurrentlyIgnored(_ context.Context, id string) bool {
	a.notFoundDevicesMu.Lock()
	defer a.notFoundDevicesMu.Unlock()

	timeOfNextAllowedRetry, ok := a.notFoundDevices[id]
	if !ok {
		return false
	}

	if time.Now().UTC().After(timeOfNextAllowedRetry) {
		delete(a.notFoundDevices, id)
		return false
	}

	return true
}

func (a *app) createUnknownDevice(ctx context.Context, se types.Event) error {
	log := logging.GetFromContext(ctx)

	d := dmtypes.Device{
		Active:      false,
		DeviceID:    DeterministicGUID(se.DevEUI),
		SensorID:    se.DevEUI,
		Name:        se.Name,
		Description: se.SensorType,
		Location: dmtypes.Location{
			Latitude:  se.Location.Latitude,
			Longitude: se.Location.Longitude,
		},
		DeviceProfile: dmtypes.DeviceProfile{
			Name:    UNKNOWN,
			Decoder: UNKNOWN,
		},
		Tenant: a.createUnknownDeviceTenant,
	}

	if len(se.Tags) > 0 {
		for k, v := range se.Tags {
			d.Tags = append(d.Tags, dmtypes.Tag{
				Name: fmt.Sprintf("%s=%s", k, v),
			})
		}
	}

	err := a.client.CreateDevice(ctx, d)
	if err != nil {
		if errors.Is(err, client.ErrDeviceExist) {
			return nil
		}

		return err
	}

	log.Debug("new device created", "sensor_id", se.DevEUI, "device_id", d.DeviceID, "name", d.Name, "tenant", d.Tenant)

	return nil
}

func (a *app) sendStatusMessage(ctx context.Context, device dmc.Device, evt *types.Event, p types.SensorPayload) error {
	log := logging.GetFromContext(ctx)

	ts := time.Now().UTC()

	if evt != nil && !evt.Timestamp.IsZero() {
		ts = evt.Timestamp.UTC()
	}

	msg := types.StatusMessage{
		DeviceID:  device.ID(),
		Tenant:    device.Tenant(),
		Timestamp: ts,
		Messages:  []string{},
	}

	if evt != nil && evt.TX != nil {
		msg.DR = &evt.TX.DR
		msg.Frequency = &evt.TX.Frequency
		msg.SpreadingFactor = &evt.TX.SpreadingFactor
	}

	if evt != nil && evt.RX != nil {
		msg.RSSI = &evt.RX.RSSI
		msg.LoRaSNR = &evt.RX.LoRaSNR
	}

	if evt != nil && evt.Status != nil {
		if !evt.Status.BatteryLevelUnavailable {
			msg.BatteryLevel = &evt.Status.BatteryLevel
		}
	}

	if p != nil {
		if msg.BatteryLevel == nil {
			if bat := p.BatteryLevel(); bat != nil {
				bat := *p.BatteryLevel()
				f := float64(bat)
				msg.BatteryLevel = &f
			}
		}

		_, messages := p.Error()

		if len(messages) > 0 {
			msg.Messages = append(msg.Messages, messages...)
		}
	}

	if evt != nil && evt.Error != nil {
		msg.Code = &evt.Error.Type
		if evt.Error.Message != "" {
			msg.Messages = append(msg.Messages, evt.Error.Message)
		}
	}

	log.Debug("publish device-status message", slog.String("device_id", msg.DeviceID), slog.Any("status", msg))

	err := a.msgCtx.PublishOnTopic(ctx, &msg)
	if err != nil {
		log.Error("failed to publish status message", "err", err.Error())
		return err
	}

	return nil
}

func DeterministicGUID(input string) string {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	hash := hasher.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], hash[:16])

	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
