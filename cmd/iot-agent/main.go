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

	forwardingEndpoint := env.GetVariableOrDie(logger, "MSG_FWD_ENDPOINT", "endpoint that incoming packages should be forwarded to")

	dmClient := createDeviceManagementClientOrDie(ctx)
	mqttClient := createMQTTClientOrDie(ctx, forwardingEndpoint, "")
	storage := createStorageOrDie(ctx)

	msgCfg := messaging.LoadConfiguration(serviceName, logger)
	initMsgCtx := func() (messaging.MsgContext, error) {
		return messaging.Initialize(msgCfg)
	}

	facade := env.GetVariableOrDefault(logger, "APPSERVER_FACADE", "chirpstack")
	svcAPI, err := initialize(ctx, facade, forwardingEndpoint, dmClient, initMsgCtx, storage)

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to setup iot agent")
	}

	mqttClient.Start()
	defer mqttClient.Stop()

	schneiderEnabled := env.GetVariableOrDefault(logger, "SCHNEIDER_ENABLED", "false")

	if schneiderEnabled == "true" {
		schneiderClient := createMQTTClientOrDie(ctx, forwardingEndpoint+"/schneider", "SCHNEIDER_")
		schneiderClient.Start()
		defer schneiderClient.Stop()
	}

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

func createMQTTClientOrDie(ctx context.Context, forwardingEndpoint, prefix string) mqtt.Client {
	mqttConfig, err := mqtt.NewConfigFromEnvironment(prefix)
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

func createStorageOrDie(ctx context.Context) storage.Storage {
	log := logging.GetFromContext(ctx)
	cfg := storage.LoadConfiguration(ctx)

	s, err := storage.Connect(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to database")
	}

	err = s.Initialize(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	return s
}

func initialize(ctx context.Context, facade, forwardingEndpoint string, dmc devicemgmtclient.DeviceManagementClient, initMsgCtx func() (messaging.MsgContext, error), storage storage.Storage) (api.API, error) {
	logger := logging.GetFromContext(ctx)

	sender := events.NewSender(ctx, initMsgCtx)
	sender.Start()

	policies, err := os.Open(opaFilePath)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to open opa policy file")
	}
	defer policies.Close()

	app := iotagent.New(dmc, sender, storage)

	r := chi.NewRouter()
	a := api.New(ctx, r, facade, forwardingEndpoint, app, policies)

	metrics.AddHandlers(r)

	return a, nil
}
