package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"testing"

	"github.com/matryer/is"
)

func withCleanFlags(t *testing.T, args []string) {
	t.Helper()

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	os.Args = args
}

// HARM-004: locks all current defaults, including flagMap fields that are
// currently shadowed by storage.LoadConfiguration (dbHost..dbSSLMode).
// Removing a field requires proving it has no effect.
func TestDefaultFlags(t *testing.T) {
	is := is.New(t)

	flags := defaultFlags()

	expected := map[flagType]string{
		listenAddress:              "0.0.0.0",
		servicePort:                "8080",
		controlPort:                "8000",
		policiesFile:               "/opt/diwise/config/authz.rego",
		dbHost:                     "",
		dbUser:                     "",
		dbPassword:                 "",
		dbPort:                     "5432",
		dbName:                     "diwise",
		dbSSLMode:                  "disable",
		createUnknownDeviceEnabled: "false",
		createUnknownDeviceTenant:  "default",
		deviceprofileFile:          "/opt/diwise/config/deviceprofiles.yaml",
		forwardingEndpoint:         "http://127.0.0.1/api/v0/messages",
		appServerFacade:            "servanet",
		logLevel:                   "debug",
		devmode:                    "false",
	}

	is.Equal(len(flags), len(expected))
	for key, want := range expected {
		is.Equal(flags[key], want)
	}

	// These keys have no default entry at all and read as "" until
	// set via env. Characterization: do not rely on their presence.
	for _, key := range []flagType{devMgmtUrl, oauth2TokenUrl, oauth2ClientId, oauth2ClientSecret} {
		_, present := flags[key]
		is.True(!present)
		is.Equal(flags[key], "")
	}
}

// HARM-004: locks env override precedence over defaults.
func TestEnvOverrides(t *testing.T) {
	is := is.New(t)
	withCleanFlags(t, []string{"iot-agent"})

	t.Setenv("LISTEN_ADDRESS", "127.0.0.1")
	t.Setenv("SERVICE_PORT", "9090")
	t.Setenv("CONTROL_PORT", "9001")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("APPSERVER_FACADE", "chirpstack")
	t.Setenv("CREATE_UNKNOWN_DEVICE_ENABLED", "true")
	t.Setenv("DEV_MGMT_URL", "http://dm:8080")

	_, flags := parseExternalConfig(context.Background(), defaultFlags())

	is.Equal(flags[listenAddress], "127.0.0.1")
	is.Equal(flags[servicePort], "9090")
	is.Equal(flags[controlPort], "9001")
	is.Equal(flags[logLevel], "info")
	is.Equal(flags[appServerFacade], "chirpstack")
	is.Equal(flags[createUnknownDeviceEnabled], "true")
	is.Equal(flags[devMgmtUrl], "http://dm:8080")
}

// HARM-004: locks CLI-over-env precedence.
func TestCLIOverridesEnv(t *testing.T) {
	is := is.New(t)
	withCleanFlags(t, []string{"iot-agent", "-loglevel=error", "-devmode=true", "-policies=/tmp/p.rego"})

	t.Setenv("LOG_LEVEL", "info")

	_, flags := parseExternalConfig(context.Background(), defaultFlags())

	is.Equal(flags[logLevel], "error")
	is.Equal(flags[devmode], "true")
	is.Equal(flags[policiesFile], "/tmp/p.rego")
}

// HARM-004: locks log level parsing, including the silent debug fallback.
func TestParseLogLevel(t *testing.T) {
	is := is.New(t)

	is.Equal(parseLogLevel("debug"), slog.LevelDebug)
	is.Equal(parseLogLevel("info"), slog.LevelInfo)
	is.Equal(parseLogLevel("warn"), slog.LevelWarn)
	is.Equal(parseLogLevel("warning"), slog.LevelWarn)
	is.Equal(parseLogLevel("error"), slog.LevelError)
	is.Equal(parseLogLevel("INFO"), slog.LevelInfo)
	is.Equal(parseLogLevel("bogus"), slog.LevelDebug)
}
