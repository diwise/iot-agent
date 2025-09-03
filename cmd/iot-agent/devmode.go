package main

import (
	"context"

	"github.com/diwise/iot-agent/internal/pkg/application"
	devicemgmtclient "github.com/diwise/iot-device-mgmt/pkg/client"
	t "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/iot-device-mgmt/pkg/types"
)

type devmodeDeviceMgmtClient struct{}

func (d *devmodeDeviceMgmtClient) FindDeviceFromDevEUI(ctx context.Context, devEUI string) (devicemgmtclient.Device, error) {
	device := t.DeviceMock{
		IDFunc: func() string {
			return application.DeterministicGUID(devEUI)
		},
		SensorTypeFunc: func() string {
			return "qalcosonic"
		},
		TenantFunc: func() string {
			return "default"
		},
	}

	return &device, nil
}

func (d *devmodeDeviceMgmtClient) FindDeviceFromInternalID(ctx context.Context, deviceID string) (devicemgmtclient.Device, error) {
	return nil, nil
}
func (d *devmodeDeviceMgmtClient) Close(ctx context.Context) {
}

func (d *devmodeDeviceMgmtClient) CreateDevice(ctx context.Context, device types.Device) error {
	return nil
}

func (d *devmodeDeviceMgmtClient) GetDeviceProfile(ctx context.Context, profileID string) (types.DeviceProfile, error) {
	return types.DeviceProfile{}, nil
}
