package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/iot-agent/internal/pkg/presentation/api"
	devicemgmtclient "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	k8shandlers "github.com/diwise/service-chassis/pkg/infrastructure/net/http/handlers"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
)

const serviceName string = "iot-agent"

func defaultFlags() flagMap {
	return flagMap{
		listenAddress: "0.0.0.0",
		servicePort:   "8080",
		controlPort:   "8000",

		policiesFile: "/opt/diwise/config/authz.rego",

		dbHost:     "",
		dbUser:     "",
		dbPassword: "",
		dbPort:     "5432",
		dbName:     "diwise",
		dbSSLMode:  "disable",

		createUnknownDeviceEnabled: "false",
		createUnknownDeviceTenant:  "default",

		forwardingEndpoint: "http://127.0.0.1/api/v0/messages",
		appServerFacade:    "servanet",

		devmode: "false",
	}
}

func main() {
	ctx, flags := parseExternalConfig(context.Background(), defaultFlags())

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(ctx, serviceName, serviceVersion, "json")
	defer cleanup()

	mqttConfig, err := mqtt.NewConfigFromEnvironment("")
	exitIf(err, logger, "mqtt configuration error")

	appCfg := appConfig{
		mqttCfg:      mqttConfig,
		messengerCfg: messaging.LoadConfiguration(ctx, serviceName, logger),
		storageCfg:   storage.LoadConfiguration(ctx),
	}

	runner, err := initialize(ctx, flags, &appCfg)
	exitIf(err, logger, "failed to initialize service runner")

	err = runner.Run(ctx)
	exitIf(err, logger, "failed to start service runner")
}

func initialize(ctx context.Context, flags flagMap, cfg *appConfig) (servicerunner.Runner[appConfig], error) {
	logger := logging.GetFromContext(ctx)

	probes := map[string]k8shandlers.ServiceProber{
		"rabbitmq":  func(context.Context) (string, error) { return "ok", nil },
		"timescale": func(context.Context) (string, error) { return "ok", nil },
		"mqtt":      func(context.Context) (string, error) { return "ok", nil },
	}

	var dmClient devicemgmtclient.DeviceManagementClient
	var messenger messaging.MsgContext
	var mqttClient mqtt.Client
	var store storage.Storage
	var facade facades.EventFunc

	_, runner := servicerunner.New(ctx, *cfg,
		webserver("control", listen(flags[listenAddress]), port(flags[controlPort]),
			pprof(), liveness(func() error { return nil }), readiness(probes),
		),
		webserver("public", listen(flags[listenAddress]), port(flags[servicePort]), tracing(true),
			muxinit(func(ctx context.Context, identifier string, port string, appCfg *appConfig, handler *http.ServeMux) error {
				logger.Debug("Initializing public webserver")

				app := application.New(
					dmClient,
					messenger,
					store,
					flags[createUnknownDeviceEnabled] == "true",
					flags[createUnknownDeviceTenant],
				)

				api.RegisterHandlers(ctx, handler, app, facade)

				return nil
			}),
		),
		oninit(func(ctx context.Context, ac *appConfig) error {
			logger.Debug("Initializing servicerunner")

			var err error

			store, err = storage.New(ctx, ac.storageCfg)
			if err != nil {
				return fmt.Errorf("failed to create storage: %w", err)
			}

			mqttClient, err = mqtt.NewClient(ctx, ac.mqttCfg, flags[forwardingEndpoint])
			if err != nil {
				return fmt.Errorf("failed to create mqtt client: %w", err)
			}

			messenger, err = messaging.Initialize(ctx, ac.messengerCfg)
			if err != nil {
				return fmt.Errorf("failed to init messenger: %w", err)
			}

			dmClient, err = devicemgmtclient.New(ctx, flags[devMgmtUrl], flags[oauth2TokenUrl], true, flags[oauth2ClientId], flags[oauth2ClientSecret])
			if err != nil {
				return fmt.Errorf("failed to create device management client: %w", err)
			}

			facade = facades.New(flags[appServerFacade])

			return nil
		}),
		onstarting(func(ctx context.Context, appCfg *appConfig) (err error) {
			logger.Debug("Starting servicerunner")
			messenger.Start()
			mqttClient.Start()

			return nil
		}),
		onshutdown(func(ctx context.Context, appCfg *appConfig) error {
			logger.Debug("Shutting down servicerunner")
			mqttClient.Stop()
			messenger.Close()
			dmClient.Close(ctx)
			store.Close()

			return nil
		}),
	)

	return runner, nil
}

func parseExternalConfig(ctx context.Context, flags flagMap) (context.Context, flagMap) {
	// Allow environment variables to override certain defaults
	envOrDef := env.GetVariableOrDefault
	flags[listenAddress] = envOrDef(ctx, "LISTEN_ADDRESS", flags[listenAddress])
	flags[controlPort] = envOrDef(ctx, "CONTROL_PORT", flags[controlPort])
	flags[servicePort] = envOrDef(ctx, "SERVICE_PORT", flags[servicePort])

	flags[policiesFile] = envOrDef(ctx, "POLICIES_FILE", flags[policiesFile])

	flags[dbHost] = envOrDef(ctx, "POSTGRES_HOST", flags[dbHost])
	flags[dbPort] = envOrDef(ctx, "POSTGRES_PORT", flags[dbPort])
	flags[dbName] = envOrDef(ctx, "POSTGRES_DBNAME", flags[dbName])
	flags[dbUser] = envOrDef(ctx, "POSTGRES_USER", flags[dbUser])
	flags[dbPassword] = envOrDef(ctx, "POSTGRES_PASSWORD", flags[dbPassword])
	flags[dbSSLMode] = envOrDef(ctx, "POSTGRES_SSLMODE", flags[dbSSLMode])

	flags[createUnknownDeviceEnabled] = envOrDef(ctx, "CREATE_UNKNOWN_DEVICE_ENABLED", flags[createUnknownDeviceEnabled])
	flags[createUnknownDeviceTenant] = envOrDef(ctx, "CREATE_UNKNOWN_DEVICE_TENANT", flags[createUnknownDeviceTenant])
	flags[forwardingEndpoint] = envOrDef(ctx, "MSG_FWD_ENDPOINT", flags[forwardingEndpoint])
	flags[appServerFacade] = envOrDef(ctx, "APPSERVER_FACADE", flags[appServerFacade])
	flags[devMgmtUrl] = envOrDef(ctx, "DEV_MGMT_URL", flags[devMgmtUrl])

	flags[oauth2TokenUrl] = envOrDef(ctx, "OAUTH2_TOKEN_URL", flags[oauth2TokenUrl])
	flags[oauth2ClientId] = envOrDef(ctx, "OAUTH2_CLIENT_ID", flags[oauth2ClientId])
	flags[oauth2ClientSecret] = envOrDef(ctx, "OAUTH2_CLIENT_SECRET", flags[oauth2ClientSecret])

	apply := func(f flagType) func(string) error {
		return func(value string) error {
			flags[f] = value
			return nil
		}
	}

	// Allow command line arguments to override defaults and environment variables
	flag.Func("policies", "an authorization policy file", apply(policiesFile))
	flag.Func("devmode", "enable dev mode", apply(devmode))
	flag.Parse()

	return ctx, flags
}

func exitIf(err error, logger *slog.Logger, msg string, args ...any) {
	if err != nil {
		logger.With(args...).Error(msg, "err", err.Error())
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}
