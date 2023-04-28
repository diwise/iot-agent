package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/farshidtz/senml/v2"
	"github.com/go-chi/chi/v5"
	"github.com/matryer/is"

	"github.com/diwise/iot-agent/internal/pkg/application"
)

func TestHealthEndpointReturns204StatusNoContent(t *testing.T) {
	is, a, _ := testSetup(t)

	server := httptest.NewServer(a.r)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodGet, server.URL+"/health", nil)
	is.Equal(resp.StatusCode, http.StatusNoContent)
}

func TestSchneiderHandler(t *testing.T) {
	is, api, app := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	api.forwardingEndpoint = server.URL + "/api/v0/messages"

	resp, _ := testRequest(is, http.MethodPost, api.forwardingEndpoint+"/schneider", bytes.NewBuffer([]byte(schneiderData)))
	is.Equal(resp.StatusCode, http.StatusOK)                 // status code should be 200
	is.Equal(len(app.HandleSensorMeasurementListCalls()), 9) // should be 9 - once for each object in schneider data
}

func TestThatApiCallsMessageReceivedProperlyOnValidMessageFromMQTT(t *testing.T) {
	is, api, app := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodPost, server.URL+"/api/v0/messages", bytes.NewBuffer([]byte(msgfromMQTT)))
	is.Equal(resp.StatusCode, http.StatusCreated)
	is.Equal(len(app.HandleSensorEventCalls()), 1)
}

func TestSenMLPayload(t *testing.T) {
	is, api, app := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodPost, server.URL+"/api/v0/messages/lwm2m", bytes.NewBuffer([]byte(senMLPayload)))
	is.Equal(resp.StatusCode, http.StatusCreated)
	is.Equal(len(app.HandleSensorMeasurementListCalls()), 1)
}

func testSetup(t *testing.T) (*is.I, *api, *iotagent.AppMock) {
	is := is.New(t)
	r := chi.NewRouter()

	app := &iotagent.AppMock{
		HandleSensorEventFunc: func(ctx context.Context, se application.SensorEvent) error {
			return nil
		},
		HandleSensorMeasurementListFunc: func(ctx context.Context, deviceID string, pack senml.Pack) error {
			return nil
		},
	}

	a := newAPI(context.Background(), r, "chirpstack", "", app)

	return is, a, app
}

func testRequest(is *is.I, method, url string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, url, body)
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

const senMLPayload string = `[{"bn": "urn:oma:lwm2m:ext:3303", "bt": 1677079794, "n": "0", "vs": "net:serva:iot:a81758fffe051d02"}, {"n": "5700", "v": -4.5}, {"u": "lat", "v": 62.36956}, {"u": "lon", "v": 17.31984}, {"n": "env", "vs": "air"}, {"n": "tenant", "vs": "default"}]`
const msgfromMQTT string = `{"level":"info","service":"iot-agent","version":"","mqtt-host":"iot.serva.net","timestamp":"2022-03-28T14:39:11.695538+02:00","message":"received payload: {\"applicationID\":\"8\",\"applicationName\":\"Water-Temperature\",\"deviceName\":\"sk-elt-temp-16\",\"deviceProfileName\":\"Elsys_Codec\",\"deviceProfileID\":\"xxxxxxxxxxxx\",\"devEUI\":\"xxxxxxxxxxxxxx\",\"rxInfo\":[{\"gatewayID\":\"xxxxxxxxxxx\",\"uplinkID\":\"xxxxxxxxxxx\",\"name\":\"SN-LGW-047\",\"time\":\"2022-03-28T12:40:40.653515637Z\",\"rssi\":-105,\"loRaSNR\":8.5,\"location\":{\"latitude\":62.36956091265246,\"longitude\":17.319844410529534,\"altitude\":0}}],\"txInfo\":{\"frequency\":867700000,\"dr\":5},\"adr\":true,\"fCnt\":10301,\"fPort\":5,\"data\":\"Bw2KDADB\",\"object\":{\"externalTemperature\":19.3,\"vdd\":3466},\"tags\":{\"Location\":\"Vangen\"}}"}`
const schneiderData string = `[{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VV1_EM01-T2/Value",
	"value":"8",
	"unit":"°C",
	"description":"Returtemperatur Varmvatten"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_OUTDOOR-TEMP/Value",
	"value":"2.1800000667572021",
	"unit":"°C",
	"description":"Utetemperatur"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VP1_EM01-ENERGY/Value",
	"value":"3372000000",
	"unit":"Wh",
	"description":"Mätarställning Värme Primär"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VP1_EM01-POWER/Value",
	"value":"66000",
	"unit":"W",
	"description":"Momentaneffekt Värme Primär"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VP1_EM01-T1/Value",
	"value":"76",
	"unit":"°C",
	"description":"Tilloppstemperatur Värme Primär"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VP1_EM01-T2/Value",
	"value":"25",
	"unit":"°C",
	"description":"Returtemperatur Värme Primär"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VV1_EM01-ENERGY/Value",
	"value":"215000000",
	"unit":"Wh",
	"description":"Mätarställning Varmvatten"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VV1_EM01-POWER/Value",
	"value":"12000",
	"unit":"W",
	"description":"Momentaneffekt Varmvatten"
	},{
	"name":"/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Framåt/UC_FRAMÅT_VV1_EM01-T1/Value",
	"value":"53",
	"unit":"°C",
	"description":"Tilloppstemperatur Varmvatten"
	}]`
