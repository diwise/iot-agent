package main

import (
	"context"

	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	"github.com/diwise/iot-agent/internal/pkg/presentation/api"
	devicemgmtclient "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

const serviceName string = "iot-agent"

func main() {

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	forwardingEndpoint := env.GetVariableOrDie(logger, "MSG_FWD_ENDPOINT", "endpoint that incoming packages should be forwarded to")
	deviceMgmtClientURL := env.GetVariableOrDie(logger, "DEV_MGMT_URL", "device management client URL")

	apiPort := env.GetVariableOrDefault(logger, "SERVICE_PORT", "8080")

	tokenURL := env.GetVariableOrDie(logger, "OAUTH2_TOKEN_URL", "a valid oauth2 token URL")
	clientID := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_ID", "a valid oauth2 client id")
	clientSecret := env.GetVariableOrDie(logger, "OAUTH2_CLIENT_SECRET", "a valid oauth2 client secret")

	app, err := SetupIoTAgent(ctx, logger, serviceName, deviceMgmtClientURL, tokenURL, clientID, clientSecret)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to setup iot agent")
	}

	mqttConfig, err := mqtt.NewConfigFromEnvironment()
	if err != nil {
		logger.Fatal().Err(err).Msg("mqtt configuration error")
	}

	mqttClient, err := mqtt.NewClient(logger, mqttConfig, forwardingEndpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create mqtt client")
	}
	mqttClient.Start()
	defer mqttClient.Stop()

	SetupAndRunApi(logger, app, apiPort)
}

func SetupIoTAgent(ctx context.Context, logger zerolog.Logger, serviceName, deviceMgmtClientURL, oauth2TokenURL, oauth2ClientID, oauth2ClientSecret string) (iotagent.IoTAgent, error) {
	dmc, err := devicemgmtclient.New(ctx, deviceMgmtClientURL, oauth2TokenURL, oauth2ClientID, oauth2ClientSecret)
	if err != nil {
		return nil, err
	}

	event := events.NewEventSender(serviceName, logger)
	event.Start()

	return iotagent.NewIoTAgent(dmc, event), nil
}

func SetupAndRunApi(logger zerolog.Logger, app iotagent.IoTAgent, port string) {
	r := chi.NewRouter()

	a := api.NewApi(logger, r, app)

	metrics.AddHandlers(r)

	a.Start(port)
}
