package api

import (
	"bytes"
	"context"
	"encoding/json"
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

	im := types.IncomingMessage{
		ID:     "123",
		Type:   "up",
		Source: "/topic/456/up",
		Data:   []byte(msgfromMQTT),
	}

	b, _ := json.Marshal(im)

	resp, _ := testRequest(is, http.MethodPost, server.URL+"/api/v0/messages", bytes.NewBuffer(b))
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
		HandleSensorEventFunc:           func(ctx context.Context, se types.Event) error { return nil },
		HandleSensorMeasurementListFunc: func(ctx context.Context, deviceID string, pack senml.Pack) error { return nil },
	}

	mux := http.NewServeMux()
	RegisterHandlers(context.Background(), mux, app, facades.New("servanet"), bytes.NewReader([]byte(policy)))

	return is, app, mux
}

func testRequest(_ *is.I, method, url string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, url, body)
	//	req.Header.Set("Authorization", "Bearer token")
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

const senMLPayload string = `[{"bn": "urn:oma:lwm2m:ext:3303", "bt": 1677079794, "n": "0", "vs": "net:serva:iot:a81758fffe051d02"}, {"n": "5700", "v": -4.5}, {"u": "lat", "v": 62.36956}, {"u": "lon", "v": 17.31984}, {"n": "env", "vs": "air"}, {"n": "tenant", "vs": "default"}]`
const msgfromMQTT string = `{"applicationID":"102","applicationName":"3_IoT-For-Klimat","deviceName":"TLD-01","deviceProfileName":"Milesight EM400TLD","deviceProfileID":"c70ad992-b55e-4f40-804b-2ebfec18ac58","devEUI":"24e124329e090021","rxInfo":[{"gatewayID":"24e124fffef477f6","uplinkID":"68bae122-70fd-4487-866d-49ccf45e9ab4","name":"SN-LGW-047","time":"2025-04-10T11:44:01.912259Z","rssi":-110,"loRaSNR":-5.8,"location":{"latitude":62.36951,"longitude":17.32014,"altitude":273}},{"gatewayID":"fcc23dfffe0a752b","uplinkID":"058471f1-3076-47a3-9ef1-6d1ad5bd248f","name":"SN-LGW-001","rssi":-104,"loRaSNR":1.2,"location":{"latitude":62.39466886148298,"longitude":17.34076023101807,"altitude":0}}],"txInfo":{"frequency":868500000,"dr":5},"adr":true,"fCnt":45797,"fPort":85,"data":"AXVXA2c4AASCXAgFAAA=","object":{"battery":87,"distance":2140,"position":"normal","temperature":5.6},"tags":{"x_typ":"mr_hushall"}}`
const policy string = `
package example.authz

# See https://www.openpolicyagent.org/docs/latest/policy-reference/ to learn more about rego

default allow := false

allow = response {
	response := {
		"tenants": ["default"]
	}
}`
