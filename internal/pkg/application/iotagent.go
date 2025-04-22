package application

import (
	"context"
	"crypto/sha1"
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

func (a *app) HandleSensorEvent(ctx context.Context, se types.Event) error {
	log := logging.GetFromContext(ctx).With(slog.String("devEUI", se.DevEUI))

	device, err := a.findDevice(ctx, se.DevEUI, a.client.FindDeviceFromDevEUI)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Warn("blacklisted", "deviceName", se.Name)
			return nil
		}

		if a.createUnknownDeviceEnabled {
			err := a.createUnknownDevice(ctx, se)
			if err != nil {
				log.Error("could not create unknown device", "err", err.Error())
				return err
			}
		}

		return err
	}

	log = log.With(slog.String("device_id", device.ID()), slog.String("type", device.SensorType()))
	ctx = logging.NewContextWithLogger(ctx, log)

	if a.createUnknownDeviceEnabled && device.SensorType() == UNKNOWN {
		a.ignoreDeviceFor(se.DevEUI, 1*time.Hour)
		return nil
	}

	var errs []error
	var payload types.SensorPayload

	if se.Payload != nil {
		decoder, converter, ok := a.registry.Get(ctx, device.SensorType())
		if !ok {
			log.Debug("no decoder found for device type", "device_type", device.SensorType())
			return nil
		}

		payload, err = decoder(ctx, se)
		if err != nil {
			log.Error("could not decode payload", "err", err.Error())
			return err
		}

		objects, err := converter(ctx, device.ID(), payload, se.Timestamp)
		if err != nil {
			log.Error("could not convert payload to objects", "err", err.Error())
			return err
		}

		if device.IsActive() {
			types := device.Types()

			for _, obj := range objects {
				if !slices.Contains(types, obj.ObjectURN()) {
					continue
				}

				err := a.handleSensorMeasurementList(ctx, lwm2m.ToPack(obj))
				if err != nil {
					log.Error("could not handle measurement", "err", err.Error())
					errs = append(errs, err)
					continue
				}
			}
		}
	}

	err = a.sendStatusMessage(ctx, device, se, payload)
	if err != nil {
		log.Warn("failed to send status message", "err", err.Error())
	}

	return errors.Join(errs...)
}

func (a *app) HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error {
	deviceID = strings.ToLower(deviceID)

	log := logging.GetFromContext(ctx).With(slog.String("device_id", deviceID))
	ctx = logging.NewContextWithLogger(ctx, log)

	_, err := a.findDevice(ctx, deviceID, a.client.FindDeviceFromInternalID)
	if err != nil {
		if errors.Is(err, errDeviceOnBlackList) {
			log.Warn("blacklisted")
			return nil
		}

		return err
	}

	return a.handleSensorMeasurementList(ctx, pack)
}

func (a *app) handleSensorMeasurementList(ctx context.Context, pack senml.Pack) error {
	log := logging.GetFromContext(ctx)
	m := core.NewMessageReceived(pack)

	err := a.msgCtx.SendCommandTo(ctx, m, "iot-core")
	if err != nil {
		log.Error("could not send message.received to iot-core", "err", err.Error())
		return err
	}

	return nil
}

var errDeviceOnBlackList = errors.New("blacklisted")

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
	d := dmtypes.Device{
		Active:   false,
		DeviceID: DeterministicGUID(se.DevEUI),
		SensorID: se.DevEUI,
		Name:     se.Name,
		DeviceProfile: dmtypes.DeviceProfile{
			Name:    UNKNOWN,
			Decoder: UNKNOWN,
		},
		Tenant: a.createUnknownDeviceTenant,
	}

	return a.client.CreateDevice(ctx, d)
}

func (a *app) sendStatusMessage(ctx context.Context, device dmc.Device, evt types.Event, p types.SensorPayload) error {
	log := logging.GetFromContext(ctx)

	msg := types.StatusMessage{
		DeviceID:  device.ID(),
		Tenant:    device.Tenant(),
		Timestamp: evt.Timestamp,
	}

	if evt.TX != nil {
		msg.DR = &evt.TX.DR
		msg.Frequency = &evt.TX.Frequency
		msg.SpreadingFactor = &evt.TX.SpreadingFactor
	}

	if evt.RX != nil {
		msg.RSSI = &evt.RX.RSSI
		msg.LoRaSNR = &evt.RX.LoRaSNR
	}

	if evt.Status != nil {
		if !evt.Status.BatteryLevelUnavailable {
			msg.BatteryLevel = &evt.Status.BatteryLevel
		}
	}

	if p != nil && msg.BatteryLevel == nil {
		if bat := p.BatteryLevel(); bat != nil {
			bat := *p.BatteryLevel()
			f := float64(bat)
			msg.BatteryLevel = &f
		}
	}

	if evt.Error != nil {
		msg.Code = &evt.Error.Type
		if evt.Error.Message != "" {
			msg.Messages = []string{evt.Error.Message}
		}
	}

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
