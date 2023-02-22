package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/go-chi/chi/v5"
	"github.com/matryer/is"

	"github.com/diwise/iot-agent/internal/pkg/application"
)

func TestHealthEndpointReturns204StatusNoContent(t *testing.T) {
	is, a, _, _ := testSetup(t)

	server := httptest.NewServer(a.r)
	defer server.Close()

	resp, _ := testRequest(is, server, http.MethodGet, "/health", nil)
	is.Equal(resp.StatusCode, http.StatusNoContent)
}

func TestThatApiCallsMessageReceivedProperlyOnValidMessageFromMQTT(t *testing.T) {
	is, api, _, app := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	resp, _ := testRequest(is, server, http.MethodPost, "/api/v0/messages", bytes.NewBuffer([]byte(msgfromMQTT)))
	is.Equal(resp.StatusCode, http.StatusCreated)
	is.Equal(len(app.MessageReceivedCalls()), 1)
}

func TestSenMLPayload(t *testing.T) {
	is, api, sender, _ := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	resp, _ := testRequest(is, server, http.MethodPost, "/api/v0/messages/lwm2m", bytes.NewBuffer([]byte(senMLPayload)))
	is.Equal(resp.StatusCode, http.StatusCreated)
	is.Equal(len(sender.SendCalls()), 1)
}

func testSetup(t *testing.T) (*is.I, *api, *events.EventSenderMock, *iotagent.AppMock) {
	is := is.New(t)
	r := chi.NewRouter()

	sender := &events.EventSenderMock{
		SendFunc: func(ctx context.Context, m messaging.CommandMessage) error {
			return nil
		},
	}

	app := &iotagent.AppMock{
		MessageReceivedFunc: func(ctx context.Context, msg []byte, ue application.UplinkASFunc) error {
			return nil
		},
	}

	a := newAPI(context.Background(), r, "chirpstack", sender, app)

	return is, a, sender, app
}

func testRequest(is *is.I, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, ts.URL+path, body)
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

const senMLPayload string = `[{"bn": "urn:oma:lwm2m:ext:3303", "bt": 1677079794, "n": "0", "vs": "net:serva:iot:a81758fffe051d02"}, {"n": "5700", "v": -4.5}, {"u": "lat", "v": 62.36956}, {"u": "lon", "v": 17.31984}, {"n": "env", "vs": "air"}, {"n": "tenant", "vs": "default"}]`
const msgfromMQTT string = `{"level":"info","service":"iot-agent","version":"","mqtt-host":"iot.serva.net","timestamp":"2022-03-28T14:39:11.695538+02:00","message":"received payload: {\"applicationID\":\"8\",\"applicationName\":\"Water-Temperature\",\"deviceName\":\"sk-elt-temp-16\",\"deviceProfileName\":\"Elsys_Codec\",\"deviceProfileID\":\"xxxxxxxxxxxx\",\"devEUI\":\"xxxxxxxxxxxxxx\",\"rxInfo\":[{\"gatewayID\":\"xxxxxxxxxxx\",\"uplinkID\":\"xxxxxxxxxxx\",\"name\":\"SN-LGW-047\",\"time\":\"2022-03-28T12:40:40.653515637Z\",\"rssi\":-105,\"loRaSNR\":8.5,\"location\":{\"latitude\":62.36956091265246,\"longitude\":17.319844410529534,\"altitude\":0}}],\"txInfo\":{\"frequency\":867700000,\"dr\":5},\"adr\":true,\"fCnt\":10301,\"fPort\":5,\"data\":\"Bw2KDADB\",\"object\":{\"externalTemperature\":19.3,\"vdd\":3466},\"tags\":{\"Location\":\"Vangen\"}}"}`
