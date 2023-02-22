package main

import (
	"context"
	"net/http"

	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
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

const serviceName string = "iot-agent"

func main() {

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	forwardingEndpoint := env.GetVariableOrDie(logger, "MSG_FWD_ENDPOINT", "endpoint that incoming packages should be forwarded to")

	dmClient := createDeviceManagementClientOrDie(ctx)
	mqttClient := createMQTTClientOrDie(ctx, forwardingEndpoint)

	msgCfg := messaging.LoadConfiguration(serviceName, logger)
	initMsgCtx := func() (messaging.MsgContext, error) {
		return messaging.Initialize(msgCfg)
	}

	facade := env.GetVariableOrDefault(logger, "APPSERVER_FACADE", "chirpstack")
	svcAPI, err := initialize(ctx, facade, dmClient, initMsgCtx)

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to setup iot agent")
	}

	mqttClient.Start()
	defer mqttClient.Stop()

	apiPort := env.GetVariableOrDefault(logger, "SERVICE_PORT", "8080")
	logger.Info().Str("port", apiPort).Msg("starting to listen for incoming connections")
	err = http.ListenAndServe(":"+apiPort, svcAPI.Router())

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start request router")
	}
}

func createDeviceManagementClientOrDie(ctx context.Context) devicemgmtclient.DeviceManagementClient {
	logger := logging.GetFromContext(ctx)

	dmURL := env.GetVariableOrDie(logger, "DEV_MGMT_URL", "url to iot-device-mgmt")
	tokenURL := env.GetVariableOrDie(logger, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	dmClient, err := devicemgmtclient.New(ctx, dmURL, tokenURL, clientID, clientSecret)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create device managagement client")
	}

	return dmClient
}

func createMQTTClientOrDie(ctx context.Context, forwardingEndpoint string) mqtt.Client {
	mqttConfig, err := mqtt.NewConfigFromEnvironment()
	logger := logging.GetFromContext(ctx)

	if err != nil {
		logger.Fatal().Err(err).Msg("mqtt configuration error")
	}

	mqttClient, err := mqtt.NewClient(logger, mqttConfig, forwardingEndpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create mqtt client")
	}

	return mqttClient
}

func initialize(ctx context.Context, facade string, dmc devicemgmtclient.DeviceManagementClient, initMsgCtx func() (messaging.MsgContext, error)) (api.API, error) {

	event := events.NewSender(ctx, initMsgCtx)
	event.Start()

	app := iotagent.New(dmc, event)

	r := chi.NewRouter()
	a := api.New(ctx, r, facade, app)

	metrics.AddHandlers(r)

	return a, nil
}
