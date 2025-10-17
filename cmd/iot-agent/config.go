package main

import (
	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/servicerunner"
)

type flagType int
type flagMap map[flagType]string

const (
	listenAddress flagType = iota
	servicePort
	controlPort

	policiesFile

	dbHost
	dbUser
	dbPassword
	dbPort
	dbName
	dbSSLMode

	createUnknownDeviceEnabled
	createUnknownDeviceTenant
	deviceprofileFile

	forwardingEndpoint
	appServerFacade
	devMgmtUrl

	oauth2ClientId
	oauth2ClientSecret
	oauth2TokenUrl

	devmode
)

type appConfig struct {
	//	messenger  messaging.MsgContext
	//	dmClient   devicemgmtclient.DeviceManagementClient
	//	mqttClient mqtt.Client
	//	storage    storage.Storage
	//	facade     facades.EventFunc

	mqttCfg      *mqtt.Config
	messengerCfg *messaging.Config
	storageCfg   *storage.Config
	dpCfg        map[string]application.DeviceProfileConfig
}

var oninit = servicerunner.OnInit[appConfig]
var onstarting = servicerunner.OnStarting[appConfig]
var onshutdown = servicerunner.OnShutdown[appConfig]
var webserver = servicerunner.WithHTTPServeMux[appConfig]
var muxinit = servicerunner.OnMuxInit[appConfig]
var listen = servicerunner.WithListenAddr[appConfig]
var port = servicerunner.WithPort[appConfig]
var pprof = servicerunner.WithPPROF[appConfig]
var liveness = servicerunner.WithK8SLivenessProbe[appConfig]
var readiness = servicerunner.WithK8SReadinessProbes[appConfig]
var tracing = servicerunner.WithTracing[appConfig]
