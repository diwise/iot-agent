package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/internal/pkg/presentation/api"
	devicemgmtclient "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/metrics"
	"github.com/go-chi/chi/v5"
)

var opaFilePath string

const serviceName string = "iot-agent"

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	flag.StringVar(&opaFilePath, "policies", "/opt/diwise/config/authz.rego", "An authorization policy file")
	flag.Parse()

	forwardingEndpoint := env.GetVariableOrDie(ctx, "MSG_FWD_ENDPOINT", "endpoint that incoming packages should be forwarded to")

	dmClient := createDeviceManagementClientOrDie(ctx)
	defer dmClient.Close(ctx)

	mqttClient := createMQTTClientOrDie(ctx, forwardingEndpoint, "")
	storage := createStorageOrDie(ctx)

	msgCfg := messaging.LoadConfiguration(ctx, serviceName, logger)
	initMsgCtx := func() (messaging.MsgContext, error) {
		return messaging.Initialize(ctx, msgCfg)
	}

	facade := env.GetVariableOrDefault(ctx, "APPSERVER_FACADE", "chirpstack")
	svcAPI, err := initialize(ctx, facade, forwardingEndpoint, dmClient, initMsgCtx, storage)

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

func createDeviceManagementClientOrDie(ctx context.Context) devicemgmtclient.DeviceManagementClient {

	dmURL := env.GetVariableOrDie(ctx, "DEV_MGMT_URL", "url to iot-device-mgmt")
	tokenURL := env.GetVariableOrDie(ctx, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

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

func createStorageOrDie(ctx context.Context) storage.Storage {
	cfg := storage.LoadConfiguration(ctx)

	s, err := storage.Connect(ctx, cfg)
	if err != nil {
		fatal(ctx, "could not connect to database", err)
	}

	err = s.Initialize(ctx)
	if err != nil {
		fatal(ctx, "failed to initialize database", err)
	}

	return s
}

func initialize(ctx context.Context, facade, forwardingEndpoint string, dmc devicemgmtclient.DeviceManagementClient, initMsgCtx func() (messaging.MsgContext, error), storage storage.Storage) (api.API, error) {

	sender := events.NewSender(ctx, initMsgCtx)
	sender.Start()

	policies, err := os.Open(opaFilePath)
	if err != nil {
		fatal(ctx, "unable to open opa policy file", err)
	}
	defer policies.Close()

	app := iotagent.New(dmc, sender, storage)

	r := chi.NewRouter()
	a, err := api.New(ctx, r, facade, forwardingEndpoint, app, policies)
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
