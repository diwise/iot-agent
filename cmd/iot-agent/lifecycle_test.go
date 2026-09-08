package main

import (
	"context"
	"errors"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	dmctest "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

var _ mqtt.Client = &fakeMQTTClient{}

type fakeMQTTClient struct {
	startErr error
	starts   int
	stops    int
	onStop   func()
}

func (f *fakeMQTTClient) Start() error {
	f.starts++
	return f.startErr
}

func (f *fakeMQTTClient) Stop() {
	f.stops++
	if f.onStop != nil {
		f.onStop()
	}
}

func (f *fakeMQTTClient) Ready() bool { return false }

type fakeStorage struct {
	closes  int
	onClose func()
}

func (f *fakeStorage) Save(context.Context, types.Event, dmc.Device, types.SensorPayload, []lwm2m.Lwm2mObject, error) error {
	return nil
}

func (f *fakeStorage) Close() error {
	f.closes++
	if f.onClose != nil {
		f.onClose()
	}
	return nil
}

// BASE-010: the messaging loop must start before the MQTT client, and an
// MQTT start failure must abort startup.
func TestStartServicesPropagatesMQTTError(t *testing.T) {
	is := is.New(t)

	var order []string
	messenger := &messaging.MsgContextMock{
		StartFunc: func() { order = append(order, "messenger") },
	}
	mqttClient := &fakeMQTTClient{startErr: errors.New("connect failed")}

	err := startServices(messenger, mqttClient)
	is.True(err != nil)
	is.Equal(order, []string{"messenger"})
	is.Equal(mqttClient.starts, 1)
}

func TestStartServicesOK(t *testing.T) {
	is := is.New(t)

	var order []string
	messenger := &messaging.MsgContextMock{
		StartFunc: func() { order = append(order, "messenger") },
	}
	mqttClient := &fakeMQTTClient{}

	is.NoErr(startServices(messenger, mqttClient))
	is.Equal(mqttClient.starts, 1)
}

// BASE-010: shutdown must stop background work, transports and storage
// exactly once, in app -> mqtt -> messenger -> device management ->
// storage order, even when invoked twice.
func TestShutdownIsOrderedAndIdempotent(t *testing.T) {
	is := is.New(t)

	var order []string
	app := &application.AppMock{
		StopFunc: func() { order = append(order, "app") },
	}
	mqttClient := &fakeMQTTClient{onStop: func() { order = append(order, "mqtt") }}
	messenger := &messaging.MsgContextMock{
		CloseFunc: func() { order = append(order, "messenger") },
	}
	dmClient := &dmctest.DeviceManagementClientMock{
		CloseFunc: func(context.Context) { order = append(order, "dm") },
	}
	store := &fakeStorage{onClose: func() { order = append(order, "storage") }}

	owned := &ownedResources{
		app:        app,
		mqttClient: mqttClient,
		messenger:  messenger,
		dmClient:   dmClient,
		store:      store,
	}

	ctx := context.Background()
	owned.shutdown(ctx)
	owned.shutdown(ctx)

	is.Equal(order, []string{"app", "mqtt", "messenger", "dm", "storage"})
	is.Equal(mqttClient.stops, 1)
	is.Equal(store.closes, 1)
	is.Equal(len(app.StopCalls()), 1)
}

// BASE-010: shutdown with no initialized resources (e.g. failed OnInit)
// must be a safe no-op.
func TestShutdownWithoutResourcesIsSafe(t *testing.T) {
	owned := &ownedResources{}

	owned.shutdown(context.Background())
	owned.shutdown(context.Background())
}

// BASE-011: readiness stubs always report OK without touching any
// dependency, including when integrations are disabled or failing.
func TestReadinessStubsAlwaysOK(t *testing.T) {
	is := is.New(t)

	probes := readinessProbes()
	is.Equal(len(probes), 3)

	for _, name := range []string{"rabbitmq", "timescale", "mqtt"} {
		status, err := probes[name](context.Background())
		is.NoErr(err)
		is.Equal(status, "ok")
	}
}
