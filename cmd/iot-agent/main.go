package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
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

	storage, err := newStorage(ctx, flags)
	exitIf(err, logger, "could not create or connect to database")

	mqttConfig, err := mqtt.NewConfigFromEnvironment("")
	exitIf(err, logger, "mqtt configuration error")

	mqttClient, err := mqtt.NewClient(ctx, mqttConfig, flags[forwardingEndpoint])
	exitIf(err, logger, "failed to create mqtt client")

	messenger, err := messaging.Initialize(ctx, messaging.LoadConfiguration(ctx, serviceName, logger))
	exitIf(err, logger, "failed to init messenger")

	dmClient, err := newDeviceManagementClient(ctx, flags)
	exitIf(err, logger, "failed to create device managagement client")

	//policies, err := os.Open(flags[policiesFile])
	//exitIf(err, logger, "unable to open opa policy file")

	appCfg := appConfig{
		messenger:  messenger,
		dmClient:   dmClient,
		mqttClient: mqttClient,
		storage:    storage,
		facade:     facades.New(flags[appServerFacade]),
	}

	runner, err := initialize(ctx, flags, &appCfg, io.NopCloser(strings.NewReader("")))
	exitIf(err, logger, "failed to initialize service runner")

	err = runner.Run(ctx)
	exitIf(err, logger, "failed to start service runner")
}

func initialize(ctx context.Context, flags flagMap, cfg *appConfig, policies io.ReadCloser) (servicerunner.Runner[appConfig], error) {
	defer policies.Close()

	probes := map[string]k8shandlers.ServiceProber{
		"rabbitmq":  func(context.Context) (string, error) { return "ok", nil },
		"timescale": func(context.Context) (string, error) { return "ok", nil },
		"mqtt":      func(context.Context) (string, error) { return "ok", nil },
	}

	_, runner := servicerunner.New(ctx, *cfg,
		webserver("control", listen(flags[listenAddress]), port(flags[controlPort]),
			pprof(), liveness(func() error { return nil }), readiness(probes),
		),
		webserver("public", listen(flags[listenAddress]), port(flags[servicePort]), tracing(true),
			muxinit(func(ctx context.Context, identifier string, port string, appCfg *appConfig, handler *http.ServeMux) error {
				app := application.New(
					appCfg.dmClient,
					appCfg.messenger,
					appCfg.storage,
					flags[createUnknownDeviceEnabled] == "true",
					flags[createUnknownDeviceTenant],
				)

				api.RegisterHandlers(ctx, handler, app, appCfg.facade, policies)

				return nil
			}),
		),
		onstarting(func(ctx context.Context, appCfg *appConfig) (err error) {
			appCfg.messenger.Start()
			appCfg.mqttClient.Start()

			return nil
		}),
		onshutdown(func(ctx context.Context, appCfg *appConfig) error {
			appCfg.mqttClient.Stop()
			appCfg.messenger.Close()
			appCfg.dmClient.Close(ctx)
			appCfg.storage.Close()

			return nil
		}),
	)

	return runner, nil
}

func newStorage(ctx context.Context, flags flagMap) (storage.Storage, error) {
	if flags[devmode] == "true" {
		return storage.NewInMemory(), nil
	}
	return storage.New(ctx, storage.LoadConfiguration(ctx))
}

func newDeviceManagementClient(ctx context.Context, flags flagMap) (devicemgmtclient.DeviceManagementClient, error) {
	if flags[devmode] == "true" {
		return &devmodeDeviceMgmtClient{}, nil
	}

	return devicemgmtclient.New(ctx, flags[devMgmtUrl], flags[oauth2TokenUrl], true, flags[oauth2ClientId], flags[oauth2ClientSecret])
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
