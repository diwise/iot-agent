package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/internal/pkg/presentation/api"
	devicemgmtclient "github.com/diwise/iot-device-mgmt/pkg/client"
	devicemgmtclientMock "github.com/diwise/iot-device-mgmt/pkg/test"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const serviceName string = "iot-agent"

var devmode bool

func main() {

	flag.BoolVar(&devmode, "devmode", false, "enable development mode")
	flag.Parse()

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	forwardingEndpoint := env.GetVariableOrDie(ctx, "MSG_FWD_ENDPOINT", "endpoint that incoming packages should be forwarded to")

	mqttClient := createMQTTClientOrDie(ctx, forwardingEndpoint, "")
	storage, err := storage.New(ctx, storage.LoadConfiguration(ctx))
	if err != nil {
		fatal(ctx, "could not create or connect to database", err)
	}

	msgCtx := createMessagingContextOrDie(ctx)
	defer msgCtx.Close()

	dmClient := createDeviceManagementClientOrDie(ctx)
	defer dmClient.Close(ctx)

	facade := env.GetVariableOrDefault(ctx, "APPSERVER_FACADE", "chirpstack")
	svcAPI, err := initialize(ctx, facade, forwardingEndpoint, dmClient, msgCtx, storage)

	if err != nil {
		fatal(ctx, "failed to setup iot agent", err)
	}

	mqttClient.Start()
	defer mqttClient.Stop()

	schneiderEnabled := env.GetVariableOrDefault(ctx, "SCHNEIDER_ENABLED", "false")

	if schneiderEnabled == "true" {
		schneiderClient := createMQTTClientOrDie(ctx, forwardingEndpoint+"/schneider", "SCHNEIDER_")
		schneiderClient.Start()
		defer schneiderClient.Stop()
	}

	apiPort := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
	logger.Info("starting to listen for incoming connections", "port", apiPort)
	err = http.ListenAndServe(":"+apiPort, svcAPI.Router())

	if err != nil {
		fatal(ctx, "failed to start request router", err)
	}
}

func createMessagingContextOrDie(ctx context.Context) messaging.MsgContext {
	log := logging.GetFromContext(ctx)

	msgCfg := messaging.LoadConfiguration(ctx, serviceName, log)
	msgCtx, err := messaging.Initialize(ctx, msgCfg)
	if err != nil {
		fatal(ctx, "failed to initialize messaging context", err)
	}

	msgCtx.Start()

	return msgCtx
}

func createDeviceManagementClientOrDie(ctx context.Context) devicemgmtclient.DeviceManagementClient {
	dmURL := env.GetVariableOrDie(ctx, "DEV_MGMT_URL", "url to iot-device-mgmt")
	tokenURL := env.GetVariableOrDie(ctx, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	if devmode {
		mock := devicemgmtclientMock.DeviceManagementClientMock{
			FindDeviceFromDevEUIFunc: func(ctx context.Context, devEUI string) (devicemgmtclient.Device, error) {
				d := &devicemgmtclientMock.DeviceMock{
					IDFunc: func() string {
						return uuid.NewString()
					},
					SensorTypeFunc: func() string {
						return "virtual"
					},
					TenantFunc: func() string {
						return "default"
					},
					IsActiveFunc: func() bool {
						return true
					},
					TypesFunc: func() []string {
						return []string{"urn:oma:lwm2m:ext:3"}
					},
				}

				return d, nil
			},
		}
		return &mock
	}

	dmClient, err := devicemgmtclient.New(ctx, dmURL, tokenURL, clientID, clientSecret)
	if err != nil {
		fatal(ctx, "failed to create device managagement client", err)
	}

	return dmClient
}

func createMQTTClientOrDie(ctx context.Context, forwardingEndpoint, prefix string) mqtt.Client {
	mqttConfig, err := mqtt.NewConfigFromEnvironment(prefix)

	if err != nil {
		fatal(ctx, "mqtt configuration error", err)
	}

	mqttClient, err := mqtt.NewClient(ctx, mqttConfig, forwardingEndpoint)
	if err != nil {
		fatal(ctx, "failed to create mqtt client", err)
	}

	return mqttClient
}

func initialize(ctx context.Context, facade, forwardingEndpoint string, dmc devicemgmtclient.DeviceManagementClient, msgCtx messaging.MsgContext, s storage.Storage) (api.API, error) {
	createUnknownDeviceEnabled := env.GetVariableOrDefault(ctx, "CREATE_UNKNOWN_DEVICE_ENABLED", "false") == "true"
	createUnknownDeviceTenant := env.GetVariableOrDefault(ctx, "CREATE_UNKNOWN_DEVICE_TENANT", "default")

	app := iotagent.New(dmc, msgCtx, createUnknownDeviceEnabled, createUnknownDeviceTenant)

	r := chi.NewRouter()
	a, err := api.New(ctx, r, facade, forwardingEndpoint, app, s)
	if err != nil {
		return nil, err
	}

	metrics.AddHandlers(r)

	return a, nil
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}
