package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sync"

	apptypes "github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	devicemgmtclient "github.com/diwise/iot-device-mgmt/pkg/client"
	test "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/iot-device-mgmt/pkg/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"gopkg.in/yaml.v3"
)

type devmodeDeviceMgmtClient struct {
	devEUI   map[string]types.Device
	deviceID map[string]types.Device
	profiles *deviceProfilesConfig
	mu       sync.Mutex
}

func newDevmodeDeviceMgmtClient(ctx context.Context) (devicemgmtclient.DeviceManagementClient, error) {
	loadDeviceProfilesConfig, err := loadDeviceProfilesConfig()
	if err != nil {
		return nil, err
	}

	c := &devmodeDeviceMgmtClient{
		devEUI:   make(map[string]types.Device),
		deviceID: make(map[string]types.Device),
		profiles: loadDeviceProfilesConfig,
	}

	devicesFile := env.GetVariableOrDefault(ctx, "DEVMODE_DEVICES_FILE", "")
	if f, err := os.Open(devicesFile); err == nil {
		defer f.Close()
		r := csv.NewReader(f)
		r.Comma = ';'
		records, err := r.ReadAll()
		if err == nil {

			for i, rec := range records {
				if i == 0 {
					continue
				}
				// devEUI;internalID;lat;lon;where;types;sensorType;name;description;active;tenant;interval;source;metadata
				d := types.Device{
					Active:   true,
					SensorID: rec[0],
					DeviceID: rec[1],
					//Latitude: rec[2],
					//Longitude: rec[3],
					//Where: rec[4],
					Lwm2mTypes: []types.Lwm2mType{},
					DeviceProfile: types.DeviceProfile{
						Decoder:  rec[6],
						Interval: 3600,
					},
					Name:        rec[7],
					Description: rec[8],
					Tenant:      rec[10],
				}

				c.devEUI[d.SensorID] = d
				c.deviceID[d.DeviceID] = d
			}
		}
	}

	return c, nil
}

func (d *devmodeDeviceMgmtClient) FindDeviceFromDevEUI(ctx context.Context, devEUI string) (devicemgmtclient.Device, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	device, ok := d.devEUI[devEUI]
	if !ok {
		return nil, devicemgmtclient.ErrNotFound
	}

	return newDeviceMock(device), nil
}

func newDeviceMock(device types.Device) devicemgmtclient.Device {
	return &test.DeviceMock{
		IsActiveFunc: func() bool {
			return device.Active
		},
		EnvironmentFunc: func() string {
			return device.Environment
		},
		IDFunc: func() string {
			return device.DeviceID
		},
		TenantFunc: func() string {
			return device.Tenant
		},
		SensorTypeFunc: func() string {
			return device.DeviceProfile.Decoder
		},
		TypesFunc: func() []string {
			t := []string{}
			for _, mt := range device.Lwm2mTypes {
				t = append(t, mt.Urn)
			}
			return t
		},
	}
}

func (d *devmodeDeviceMgmtClient) FindDeviceFromInternalID(ctx context.Context, deviceID string) (devicemgmtclient.Device, error) {
	return nil, nil
}
func (d *devmodeDeviceMgmtClient) Close(ctx context.Context) {
}

func (d *devmodeDeviceMgmtClient) CreateDevice(ctx context.Context, device types.Device) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.devEUI[device.SensorID] = device
	d.deviceID[device.DeviceID] = device

	return nil
}

func (d *devmodeDeviceMgmtClient) GetDeviceProfile(ctx context.Context, deviceProfileID string) (*types.DeviceProfile, error) {
	for _, dp := range d.profiles.DeviceProfiles {
		if dp.Name == deviceProfileID {
			return &types.DeviceProfile{
				Name:     dp.Name,
				Decoder:  dp.Decoder,
				Interval: dp.Interval,
				Types:    dp.Types,
			}, nil
		}
	}

	return nil, fmt.Errorf("not found")
}

type devmodeStorage struct {
	storage.StorageMock
}

func (d *devmodeStorage) Ping(ctx context.Context) error {
	return nil
}

func newDevmodeStorage(_ context.Context) (storage.Storage, error) {
	return &devmodeStorage{
		storage.StorageMock{
			SaveFunc: func(ctx context.Context, se apptypes.Event, device devicemgmtclient.Device, payload apptypes.SensorPayload, objects []lwm2m.Lwm2mObject, err error) error {
				return nil
			},
		},
	}, nil
}

func loadDeviceProfilesConfig() (*deviceProfilesConfig, error) {
	var cfg deviceProfilesConfig
	err := yaml.Unmarshal([]byte(deviceprofilesYaml), &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

type deviceProfilesConfig struct {
	DeviceProfiles []deviceProfile `yaml:"deviceprofiles"`
	Types          []typeInfo      `yaml:"types"`
}

type deviceProfile struct {
	Name     string   `yaml:"name"`
	Decoder  string   `yaml:"decoder"`
	Interval int      `yaml:"interval"`
	Types    []string `yaml:"types"`
}

type typeInfo struct {
	Urn  string `yaml:"urn"`
	Name string `yaml:"name"`
}

const deviceprofilesYaml = `
  deviceprofiles:
    - name: axsensor
      decoder: axsensor
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3330
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3327
        - urn:oma:lwm2m:ext:3303
    - name: elsys
      decoder: elsys
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3301
        - urn:oma:lwm2m:ext:3428
        - urn:oma:lwm2m:ext:3302
        - urn:oma:lwm2m:ext:3200
    - name: elt_2_hp
      decoder: elt_2_hp
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3301
        - urn:oma:lwm2m:ext:3428
        - urn:oma:lwm2m:ext:3302
        - urn:oma:lwm2m:ext:3200
    - name: enviot
      decoder: enviot
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3330
    - name: milesight
      decoder: milesight
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3428
        - urn:oma:lwm2m:ext:3330
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3303
    - name: niab-fls
      decoder: niab-fls
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3330
    - name: qalcosonic
      decoder: qalcosonic
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3424
        - urn:oma:lwm2m:ext:3303
    - name: senlabt
      decoder: senlabt
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
    - name: sensative
      decoder: sensative
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3302
    - name: sensefarm
      decoder: sensefarm
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3327
        - urn:oma:lwm2m:ext:3323
    - name: vegapuls_air_41
      decoder: vegapuls_air_41
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3330
        - urn:oma:lwm2m:ext:3303
    - name: airquality
      decoder: airquality
      interval: 86400
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3428
    - name: virtual
      decoder: virtual
      interval: 3600
      types:
        - urn:oma:lwm2m:ext:3
        - urn:oma:lwm2m:ext:3200
        - urn:oma:lwm2m:ext:3301
        - urn:oma:lwm2m:ext:3302
        - urn:oma:lwm2m:ext:3303
        - urn:oma:lwm2m:ext:3304
        - urn:oma:lwm2m:ext:3323
        - urn:oma:lwm2m:ext:3327
        - urn:oma:lwm2m:ext:3328
        - urn:oma:lwm2m:ext:3330
        - urn:oma:lwm2m:ext:3331
        - urn:oma:lwm2m:ext:3350
        - urn:oma:lwm2m:ext:3411
        - urn:oma:lwm2m:ext:3424
        - urn:oma:lwm2m:ext:3428
        - urn:oma:lwm2m:ext:3434
        - urn:oma:lwm2m:ext:3435

  types:
    - urn: urn:oma:lwm2m:ext:3
      name: Device
    - urn: urn:oma:lwm2m:ext:3303
      name: Temperature
    - urn: urn:oma:lwm2m:ext:3304
      name: Humidity
    - urn: urn:oma:lwm2m:ext:3301
      name: Illuminance
    - urn: urn:oma:lwm2m:ext:3428
      name: AirQuality
    - urn: urn:oma:lwm2m:ext:3302
      name: Presence
    - urn: urn:oma:lwm2m:ext:3200
      name: DigitalInput
    - urn: urn:oma:lwm2m:ext:3330
      name: Distance
    - urn: urn:oma:lwm2m:ext:3327
      name: Conductivity
    - urn: urn:oma:lwm2m:ext:3323
      name: Pressure
    - urn: urn:oma:lwm2m:ext:3435
      name: FillingLevel
    - urn: urn:oma:lwm2m:ext:3424
      name: WaterMeter
    - urn: urn:oma:lwm2m:ext:3411
      name: Battery
    - urn: urn:oma:lwm2m:ext:3434
      name: PeopleCounter
    - urn: urn:oma:lwm2m:ext:3328
      name: Power
    - urn: urn:oma:lwm2m:ext:3331
      name: Energy
    - urn: urn:oma:lwm2m:ext:3350
      name: Stopwatch
`
