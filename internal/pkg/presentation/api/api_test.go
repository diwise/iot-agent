package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"

	"github.com/matryer/is"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/senml"
)

/*
func TestHealthEndpointReturns204StatusNoContent(t *testing.T) {
	is, _, mux := testSetup(t)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodGet, server.URL+"/health", nil)
	is.Equal(resp.StatusCode, http.StatusNoContent)
}
*/

func TestThatApiCallsMessageReceivedProperlyOnValidMessageFromMQTT(t *testing.T) {
	is, app, mux := testSetup(t)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodPost, server.URL+"/api/v0/messages", bytes.NewBuffer([]byte(msgfromMQTT)))
	is.Equal(resp.StatusCode, http.StatusCreated)
	is.Equal(len(app.HandleSensorEventCalls()), 1)
}

func TestSenMLPayload(t *testing.T) {
	is, app, mux := testSetup(t)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodPost, server.URL+"/api/v0/messages/lwm2m", bytes.NewBuffer([]byte(senMLPayload)))
	is.Equal(resp.StatusCode, http.StatusCreated)
	is.Equal(len(app.HandleSensorMeasurementListCalls()), 1)
}

func testSetup(t *testing.T) (*is.I, *application.AppMock, *http.ServeMux) {
	is := is.New(t)

	app := &application.AppMock{
		HandleSensorEventFunc:           func(ctx context.Context, se types.SensorEvent) error { return nil },
		HandleSensorMeasurementListFunc: func(ctx context.Context, deviceID string, pack senml.Pack) error { return nil },
		SaveFunc:                        func(ctx context.Context, se types.SensorEvent) error { return nil },
	}

	mux := http.NewServeMux()
	RegisterHandlers(context.Background(), mux, app, facades.ChirpStack, bytes.NewReader([]byte(policy)))

	return is, app, mux
}

func testRequest(_ *is.I, method, url string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Bearer token")
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

const senMLPayload string = `[{"bn": "urn:oma:lwm2m:ext:3303", "bt": 1677079794, "n": "0", "vs": "net:serva:iot:a81758fffe051d02"}, {"n": "5700", "v": -4.5}, {"u": "lat", "v": 62.36956}, {"u": "lon", "v": 17.31984}, {"n": "env", "vs": "air"}, {"n": "tenant", "vs": "default"}]`
const msgfromMQTT string = `{"level":"info","service":"iot-agent","version":"","mqtt-host":"iot.serva.net","timestamp":"2022-03-28T14:39:11.695538+02:00","message":"received payload: {\"applicationID\":\"8\",\"applicationName\":\"Water-Temperature\",\"deviceName\":\"sk-elt-temp-16\",\"deviceProfileName\":\"Elsys_Codec\",\"deviceProfileID\":\"xxxxxxxxxxxx\",\"devEUI\":\"xxxxxxxxxxxxxx\",\"rxInfo\":[{\"gatewayID\":\"xxxxxxxxxxx\",\"uplinkID\":\"xxxxxxxxxxx\",\"name\":\"SN-LGW-047\",\"time\":\"2022-03-28T12:40:40.653515637Z\",\"rssi\":-105,\"loRaSNR\":8.5,\"location\":{\"latitude\":62.36956091265246,\"longitude\":17.319844410529534,\"altitude\":0}}],\"txInfo\":{\"frequency\":867700000,\"dr\":5},\"adr\":true,\"fCnt\":10301,\"fPort\":5,\"data\":\"Bw2KDADB\",\"object\":{\"externalTemperature\":19.3,\"vdd\":3466},\"tags\":{\"Location\":\"Vangen\"}}"}`
const policy string = `
package example.authz

# See https://www.openpolicyagent.org/docs/latest/policy-reference/ to learn more about rego

default allow := false

allow = response {
	response := {
		"tenants": ["default"]
	}
}`
