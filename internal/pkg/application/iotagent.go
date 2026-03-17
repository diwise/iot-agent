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

	notFoundDevices   map[string]time.Time
	notFoundDevicesMu sync.Mutex

	createUnknownDeviceEnabled bool
	createUnknownDeviceTenant  string
	dpCfg                      map[string]profile
	dpCfgMu                    sync.RWMutex
}

type profile struct {
	Cfg   DeviceProfileConfig
	Types []string
}

func New(dmc dmc.DeviceManagementClient, msgCtx messaging.MsgContext, storage storage.Storage, createUnknownDeviceEnabled bool, createUnknownDeviceTenant string, dpCfg map[string]DeviceProfileConfig) App {
	d := decoders.NewRegistry()

	a := &app{
		registry:                   d,
		client:                     dmc,
		msgCtx:                     msgCtx,
		store:                      storage,
		notFoundDevices:            make(map[string]time.Time),
		createUnknownDeviceEnabled: createUnknownDeviceEnabled,
		createUnknownDeviceTenant:  createUnknownDeviceTenant,
		dpCfg:                      make(map[string]profile),
	}

	for sensorType, p := range dpCfg {
		if p.Tenant == "" {
			p.Tenant = createUnknownDeviceTenant
		}
		a.dpCfg[strings.ToLower(sensorType)] = profile{
			Cfg:   p,
			Types: []string{},
		}
	}

	if _, ok := a.dpCfg[UNKNOWN]; !ok {
		a.dpCfg[UNKNOWN] = profile{
			Cfg: DeviceProfileConfig{
				ProfileName: UNKNOWN,
				Tenant:      createUnknownDeviceTenant,
				Activate:    false,
				Location:    false,
				Tags:        Tags{Enabled: false},
			},
			Types: []string{},
		}
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			a.notFoundDevicesMu.Lock()
			for devEUI, ts := range a.notFoundDevices {
				if time.Now().UTC().After(ts.UTC()) {
					delete(a.notFoundDevices, devEUI)
				}
			}
			a.notFoundDevicesMu.Unlock()
		}
	}()

	return a
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
		a.ignoreDeviceFor(se.DevEUI, 5*time.Minute)
		return device, nil, nil, types.ErrDeviceIgnored
	}

	if se.Payload == nil {
		return device, nil, nil, types.ErrPayloadContainsNoData
	}

	decoder, converter, ok := a.registry.Get(ctx, device.SensorType())
	if !ok {
		return device, nil, nil, types.ErrDecoderOrConverterNotFound
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

	if errors.Is(err, types.ErrDecoderOrConverterNotFound) {
		log.Warn("decoder or converter not found for sensor type", "sensor_type", device.SensorType())
		return nil
	}

	if errors.Is(err, types.ErrPayloadEmpty) || errors.Is(err, types.ErrPayloadContainsNoData) {
		log.Debug("event contains no payload")
		return nil
	}

	if errors.Is(err, types.ErrDeviceOnBlackList) {
		log.Debug("device is blacklisted", "sensor_id", se.DevEUI)
		return nil
	}

	if errors.Is(err, types.ErrDeviceIgnored) {
		log.Debug("device is ignored since the type is unknown", "sensor_id", se.DevEUI, slog.Bool("create_unknown_devices", a.createUnknownDeviceEnabled))
		return nil
	}

	if a.createUnknownDeviceEnabled && errors.Is(err, types.ErrDeviceNotFound) {
		err := a.createUnknownDevice(ctx, se)
		if err != nil {
			log.Error("could not create new device of unknown type", "err", err.Error())
			return err
		}

		log.Info("created new device of unknown type", "sensor_id", se.DevEUI, "tenant", a.createUnknownDeviceTenant)
		return nil
	}

	if errors.Is(err, types.ErrNoDevice) {
		log.Debug("no device found for sensor id", "sensor_id", se.DevEUI)
		return types.ErrNoDevice
	}

	if err != nil {
		log.Error("failed to decode and convert sensor event", "err", err.Error())
		return err
	}

	if err := a.sendStatusMessage(ctx, device, &se, payload); err != nil {
		log.Warn("failed to send status message", "err", err.Error())
	}

	if !device.IsActive() {
		log.Debug("device is not active", "device_id", device.ID(), "sensor_id", se.DevEUI)
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
		if errors.Is(err, types.ErrDeviceOnBlackList) {
			log.Debug("device is blacklisted", "device_id", deviceID)
			return nil
		}
		if errors.Is(err, types.ErrDeviceIgnored) {
			log.Debug("device is ignored since the type is unknown", "device_id", deviceID, slog.Bool("create_unknown_devices", a.createUnknownDeviceEnabled))
			return nil
		}

		log.Error("could not find device for id", "device_id", deviceID, "err", err.Error())

		return err
	}

	if err := a.sendStatusMessage(ctx, d, nil, nil); err != nil {
		log.Warn("failed to send status message", "err", err.Error())
	}

	if err := a.handleSensorMeasurementList(ctx, pack); err != nil {
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
		return nil, types.ErrDeviceIgnored
	}

	device, err := finder(ctx, id)
	if err != nil {
		if errors.Is(err, dmc.ErrNotFound) {
			return nil, types.ErrDeviceNotFound
		}
		return nil, err
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

type DeviceProfileConfig struct {
	ProfileName string `json:"profile_name" yaml:"profile_name"`
	Tenant      string `json:"tenant" yaml:"tenant"`
	Activate    bool   `json:"activate" yaml:"activate"`
	Location    bool   `json:"location" yaml:"location"`
	Tags        Tags   `json:"tags" yaml:"tags"`
}

type Tags struct {
	Enabled  bool              `json:"enabled"`
	Metadata bool              `json:"metadata"`
	Mappings map[string]string `json:"mappings"`
}

func (a *app) getDeviceProfileOrUnknown(ctx context.Context, sensorType string) profile {
	log := logging.GetFromContext(ctx)

	sensorType = strings.ToLower(sensorType)

	log.Debug("get device profile for sensor type", "sensor_type", sensorType)

	a.dpCfgMu.RLock()
	dp, ok := a.dpCfg[sensorType]
	unknown := a.dpCfg[UNKNOWN]
	a.dpCfgMu.RUnlock()

	if !ok {
		log.Debug("device profile not found, returning UNKNOWN", "name", dp.Cfg.ProfileName)
		return unknown
	}

	if len(dp.Types) > 0 {
		return dp
	}

	p, err := a.client.GetDeviceProfile(ctx, dp.Cfg.ProfileName)
	if err != nil {
		log.Debug("could not fetch device profile from device management, returning UNKNOWN", "name", dp.Cfg.ProfileName)
		return unknown
	}

	a.dpCfgMu.Lock()
	dp.Types = p.Types
	a.dpCfg[sensorType] = dp
	a.dpCfgMu.Unlock()

	return dp
}

func (a *app) createUnknownDevice(ctx context.Context, se types.Event) error {
	log := logging.GetFromContext(ctx)

	var d dmtypes.Device
	var s dmtypes.SensorInputModel

	p := a.getDeviceProfileOrUnknown(ctx, se.SensorType)

	s = dmtypes.SensorInputModel{
		SensorID:        se.DevEUI,
		SensorProfileID: p.Cfg.ProfileName,
		Location: &dmtypes.Location{
			Latitude:  se.Location.Latitude,
			Longitude: se.Location.Longitude,
		},
	}

	if se.Name != "" {
		name := se.Name
		s.Name = &name
	}

	err := a.client.CreateSensor(ctx, s)
	if err != nil {
		if !errors.Is(err, client.ErrConflict) {
			return err
		}
	}

	if s.SensorProfileID == UNKNOWN {
		log.Debug("we will not create a device for a sensor of unknown profile")
		return nil
	}

	d = dmtypes.Device{
		Active:      p.Cfg.Activate,
		DeviceID:    DeterministicGUID(se.DevEUI),
		SensorID:    se.DevEUI,
		Name:        se.Name,
		Description: se.SensorType,

		SensorProfile: dmtypes.SensorProfile{
			Name:    p.Cfg.ProfileName,
			Decoder: p.Cfg.ProfileName,
			Types:   p.Types,
		},

		Tenant: p.Cfg.Tenant,
	}

	if len(p.Types) > 0 {
		d.Lwm2mTypes = make([]dmtypes.Lwm2mType, 0, len(p.Types))
		for _, t := range p.Types {
			d.Lwm2mTypes = append(d.Lwm2mTypes, dmtypes.Lwm2mType{
				Urn: t,
			})
		}
	}

	if p.Cfg.Location {
		d.Location = dmtypes.Location{
			Latitude:  se.Location.Latitude,
			Longitude: se.Location.Longitude,
		}
	}

	if p.Cfg.Tags.Enabled {
		if len(se.Tags) > 0 {
			for k, v := range se.Tags {
				tag := k

				if len(v) > 0 {
					tag = fmt.Sprintf("%s=%s", k, v[0])
				}

				d.Tags = append(d.Tags, dmtypes.Tag{
					Name: tag,
				})

				if !p.Cfg.Tags.Metadata {
					continue
				}

				key := strings.ToLower(strings.TrimSpace(k))
				if p.Cfg.Tags.Mappings != nil {
					if mappedKey, ok := p.Cfg.Tags.Mappings[key]; ok {
						key = mappedKey
					}
				}

				if i := slices.IndexFunc(d.Metadata, func(m dmtypes.Metadata) bool { return m.Key == key }); i == -1 {
					d.Metadata = append(d.Metadata, dmtypes.Metadata{
						Key:   key,
						Value: v[0],
					})
				} else {
					d.Metadata[i].Value = v[0]
				}
			}
		}
	}

	err = a.client.CreateDevice(ctx, d)
	if err != nil {
		if errors.Is(err, client.ErrDeviceExist) {
			return nil
		}

		return err
	}

	if p.Cfg.ProfileName != UNKNOWN {
		a.notFoundDevicesMu.Lock()
		delete(a.notFoundDevices, se.DevEUI)
		a.notFoundDevicesMu.Unlock()
	}

	log.Debug("new device created", "sensor_id", se.DevEUI, "device_id", d.DeviceID, "profile_name", d.SensorProfile.Name, "name", d.Name, "tenant", d.Tenant)

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

		code, messages := p.Error()

		if code != "" {
			msg.Code = &code
		}

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
