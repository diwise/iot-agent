package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/internal/application"
	"github.com/diwise/iot-agent/internal/application/facades"
	"github.com/diwise/iot-agent/internal/infrastructure/mqtt"
	"github.com/diwise/iot-agent/internal/infrastructure/storage"
	"github.com/diwise/iot-agent/internal/presentation/api"
	dmclient "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	k8shandlers "github.com/diwise/service-chassis/pkg/infrastructure/net/http/handlers"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
	"gopkg.in/yaml.v3"
)

const serviceName string = "iot-agent"

func defaultFlags() flagMap {
	return flagMap{
		listenAddress: "0.0.0.0",
		servicePort:   "8080",
		controlPort:   "8000",

		createUnknownDeviceEnabled: "false",
		createUnknownDeviceTenant:  "default",
		deviceprofileFile:          "/opt/diwise/config/deviceprofiles.yaml",

		forwardingEndpoint: "http://127.0.0.1/api/v0/messages",
		appServerFacade:    "servanet",

		logLevel: "debug",

		devmode: "false",
	}
}

func main() {
	ctx, flags := parseExternalConfig(context.Background(), defaultFlags())

	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(ctx, serviceName, serviceVersion, "json")
	defer cleanup()

	logging.SetLogLevel(parseLogLevel(flags[logLevel]))

	f, err := os.Open(flags[deviceprofileFile])
	exitIf(err, logger, "failed to open device profile configuration file")

	dpCfg, err := parseExternalConfigFile(ctx, f)
	exitIf(err, logger, "failed to parse device profile configuration")

	mqttConfig, err := mqtt.NewConfigFromEnvironment("")
	exitIf(err, logger, "mqtt configuration error")

	messengerConfig := messaging.LoadConfiguration(ctx, serviceName, logger)
	storageConfig := storage.LoadConfiguration(ctx)

	appCfg := appConfig{
		mqttCfg:      &mqttConfig,
		messengerCfg: &messengerConfig,
		storageCfg:   &storageConfig,
		dpCfg:        dpCfg,
		devmode:      devmodeEnabled(flags),
	}

	runner, err := initialize(ctx, flags, &appCfg)
	exitIf(err, logger, "failed to initialize service runner")

	err = runner.Run(ctx)
	exitIf(err, logger, "failed to start service runner")
}

func initialize(ctx context.Context, flags flagMap, cfg *appConfig) (servicerunner.Runner[appConfig], error) {
	logger := logging.GetFromContext(ctx)

	var dmClient dmclient.DeviceManagementClient
	var messenger messaging.MsgContext
	var mqttClient mqtt.Client
	var store storage.Storage
	var facade facades.EventFunc
	var app application.App

	owned := &ownedResources{}

	probes := readinessProbes()

	_, runner := servicerunner.New(ctx, *cfg,
		webserver("control", listen(flags[listenAddress]), port(flags[controlPort]),
			pprof(), liveness(func() error { return nil }), readiness(probes),
		),
		webserver("public", listen(flags[listenAddress]), port(flags[servicePort]), tracing(true),
			muxinit(func(ctx context.Context, identifier string, port string, appCfg *appConfig, handler *http.ServeMux) error {
				logger.Debug("initializing public webserver")

				api.RegisterHandlers(ctx, handler, app, facade)

				return nil
			}),
		),
		oninit(func(ctx context.Context, ac *appConfig) error {
			logger.Debug("initializing servicerunner")

			var err error

			store, err = newStorage(ctx, *ac.storageCfg, ac.devmode)
			if err != nil {
				return fmt.Errorf("failed to create storage: %w", err)
			}
			owned.store = store

			mqttClient, err = mqtt.NewClient(ctx, *ac.mqttCfg, flags[forwardingEndpoint])
			if err != nil {
				owned.shutdown(ctx)
				store = nil
				return fmt.Errorf("failed to create mqtt client: %w", err)
			}
			owned.mqttClient = mqttClient

			messenger, err = messaging.Initialize(ctx, *ac.messengerCfg)
			if err != nil {
				owned.shutdown(ctx)
				store, mqttClient = nil, nil
				return fmt.Errorf("failed to init messenger: %w", err)
			}
			owned.messenger = messenger

			dmClient, err = newDeviceMgmtClient(ctx, flags[devMgmtUrl], flags[oauth2TokenUrl], flags[oauth2ClientId], flags[oauth2ClientSecret], ac.devmode)
			if err != nil {
				owned.shutdown(ctx)
				store, mqttClient, messenger = nil, nil, nil
				return fmt.Errorf("failed to create device management client: %w", err)
			}
			owned.dmClient = dmClient

			facade = facades.New(flags[appServerFacade])

			app = application.New(
				dmClient,
				messenger,
				store,
				createUnknownDevicesEnabled(flags),
				flags[createUnknownDeviceTenant],
				ac.dpCfg,
			)
			owned.app = app

			return nil
		}),
		onstarting(func(ctx context.Context, appCfg *appConfig) (err error) {
			logger.Debug("starting servicerunner")

			// OnStarting failures bypass OnShutdown in the runner, so
			// clean up acquired resources on error below.
			defer func() {
				if err != nil {
					owned.shutdown(ctx)
				}
			}()

			return startServices(messenger, mqttClient)
		}),
		onshutdown(func(ctx context.Context, appCfg *appConfig) error {
			logger.Debug("shutting down servicerunner")

			owned.shutdown(ctx)

			return nil
		}),
	)

	return runner, nil
}

// readinessProbes returns the named readiness stubs. Per harmonization
// standard they always report OK and never call any dependency. Probe
// names are preserved for external deployment definitions.
func readinessProbes() map[string]k8shandlers.ServiceProber {
	return map[string]k8shandlers.ServiceProber{
		"rabbitmq":  func(context.Context) (string, error) { return "ok", nil },
		"timescale": func(context.Context) (string, error) { return "ok", nil },
		"mqtt":      func(context.Context) (string, error) { return "ok", nil },
	}
}

// devmodeEnabled is the minimal production seam for the dev-mode
// toggle. Only the exact string "true" enables it.
func devmodeEnabled(flags flagMap) bool {
	return flags[devmode] == "true"
}

// createUnknownDevicesEnabled is the minimal production seam for the
// unknown-device toggle. Only the exact string "true" enables it;
// ParseBool spellings such as "TRUE" or "1" intentionally do not.
func createUnknownDevicesEnabled(flags flagMap) bool {
	return flags[createUnknownDeviceEnabled] == "true"
}

// startServices starts the messaging loop before the MQTT client, so no
// inbound message can arrive before the command path is running.
//
// Startup contract (REV-009): startup is complete once the loops are
// started. The real MQTT client connects asynchronously and reconnects
// in the background; connection failures are logged, never fatal.
// Only synchronous start errors abort startup.
func startServices(messenger messaging.MsgContext, mqttClient mqtt.Client) error {
	messenger.Start()

	if err := mqttClient.Start(); err != nil {
		return fmt.Errorf("failed to start mqtt client: %w", err)
	}

	return nil
}

// ownedResources tracks the resources created during OnInit so shutdown
// stops background work, then transports, then storage, exactly once.
// Shutdown is nil-safe (partial OnInit) and idempotent: the underlying
// messenger Close is not safe to call twice, hence the sync.Once guard.
type ownedResources struct {
	once       sync.Once
	app        application.App
	mqttClient mqtt.Client
	messenger  messaging.MsgContext
	dmClient   dmclient.DeviceManagementClient
	store      storage.Storage
}

func (o *ownedResources) shutdown(ctx context.Context) {
	o.once.Do(func() {
		if o.app != nil {
			o.app.Stop()
		}
		if o.mqttClient != nil {
			o.mqttClient.Stop()
		}
		if o.messenger != nil {
			o.messenger.Close()
		}
		if o.dmClient != nil {
			o.dmClient.Close(ctx)
		}
		if o.store != nil {
			if err := o.store.Close(); err != nil {
				logging.GetFromContext(ctx).Debug("failed to close storage", "err", err.Error())
			}
		}
	})
}

func newStorage(ctx context.Context, cfg storage.Config, devmode bool) (storage.Storage, error) {
	if devmode {
		logging.GetFromContext(ctx).Warn("devmode is enabled, using in-memory storage")
		return newDevmodeStorage(ctx)
	}

	return storage.New(ctx, cfg)
}

func newDeviceMgmtClient(ctx context.Context, url, tokenUrl, clientId, clientSecret string, devmode bool) (dmclient.DeviceManagementClient, error) {
	if devmode {
		logging.GetFromContext(ctx).Warn("devmode is enabled, using device management client mock")
		return newDevmodeDeviceMgmtClient(ctx)
	}

	return dmclient.New(ctx, url, tokenUrl, true, clientId, clientSecret)
}

func parseExternalConfig(ctx context.Context, flags flagMap) (context.Context, flagMap) {
	// Allow environment variables to override certain defaults
	envOrDef := env.GetVariableOrDefault
	flags[listenAddress] = envOrDef(ctx, "LISTEN_ADDRESS", flags[listenAddress])
	flags[controlPort] = envOrDef(ctx, "CONTROL_PORT", flags[controlPort])
	flags[servicePort] = envOrDef(ctx, "SERVICE_PORT", flags[servicePort])
	flags[logLevel] = envOrDef(ctx, "LOG_LEVEL", flags[logLevel])

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
	flag.Func("deviceprofiles", "a device profile configuration file", apply(deviceprofileFile))
	flag.Func("devmode", "enable dev mode", apply(devmode))
	flag.Func("loglevel", "set the log level", apply(logLevel))
	flag.Parse()

	return ctx, flags
}

func parseExternalConfigFile(_ context.Context, f io.ReadCloser) (map[string]application.DeviceProfileConfig, error) {
	defer f.Close()

	var cfg map[string]application.DeviceProfileConfig
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read device profile file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device profile file: %w", err)
	}

	return cfg, nil
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

func exitIf(err error, logger *slog.Logger, msg string, args ...any) {
	if err != nil {
		logger.With(args...).Error(msg, "err", err.Error())
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}
