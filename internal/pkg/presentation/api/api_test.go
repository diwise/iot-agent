package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"

	"github.com/go-chi/chi/v5"
	"github.com/matryer/is"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/senml"
)

func TestHealthEndpointReturns204StatusNoContent(t *testing.T) {
	is, a, _ := testSetup(t)

	server := httptest.NewServer(a.r)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodGet, server.URL+"/health", nil)
	is.Equal(resp.StatusCode, http.StatusNoContent)
}

func TestDebugPprofHeapEndpointReturns200OK(t *testing.T) {
	is, a, _ := testSetup(t)

	server := httptest.NewServer(a.r)
	defer server.Close()

	resp, _ := testRequest(is, http.MethodGet, server.URL+"/debug/pprof/heap", nil)
	is.Equal(resp.StatusCode, http.StatusOK)
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

	a, _ := newAPI(context.Background(), r, "chirpstack", "", app, bytes.NewBufferString(opaModule))

	return is, a, app
}

func testRequest(_ *is.I, method, url string, body io.Reader) (*http.Response, string) {
	req, _ := http.NewRequest(method, url, body)
	resp, _ := http.DefaultClient.Do(req)
	respBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return resp, string(respBody)
}

const senMLPayload string = `[{"bn": "urn:oma:lwm2m:ext:3303", "bt": 1677079794, "n": "0", "vs": "net:serva:iot:a81758fffe051d02"}, {"n": "5700", "v": -4.5}, {"u": "lat", "v": 62.36956}, {"u": "lon", "v": 17.31984}, {"n": "env", "vs": "air"}, {"n": "tenant", "vs": "default"}]`
const msgfromMQTT string = `{"level":"info","service":"iot-agent","version":"","mqtt-host":"iot.serva.net","timestamp":"2022-03-28T14:39:11.695538+02:00","message":"received payload: {\"applicationID\":\"8\",\"applicationName\":\"Water-Temperature\",\"deviceName\":\"sk-elt-temp-16\",\"deviceProfileName\":\"Elsys_Codec\",\"deviceProfileID\":\"xxxxxxxxxxxx\",\"devEUI\":\"xxxxxxxxxxxxxx\",\"rxInfo\":[{\"gatewayID\":\"xxxxxxxxxxx\",\"uplinkID\":\"xxxxxxxxxxx\",\"name\":\"SN-LGW-047\",\"time\":\"2022-03-28T12:40:40.653515637Z\",\"rssi\":-105,\"loRaSNR\":8.5,\"location\":{\"latitude\":62.36956091265246,\"longitude\":17.319844410529534,\"altitude\":0}}],\"txInfo\":{\"frequency\":867700000,\"dr\":5},\"adr\":true,\"fCnt\":10301,\"fPort\":5,\"data\":\"Bw2KDADB\",\"object\":{\"externalTemperature\":19.3,\"vdd\":3466},\"tags\":{\"Location\":\"Vangen\"}}"}`

const opaModule string = `
	#
	# Use https://play.openpolicyagent.org for easier editing/validation of this policy file
	#
	
	package example.authz
	
	default allow := false
	
	allow = response {
		is_valid_token
	
		input.method == "GET"
		pathstart := array.slice(input.path, 0, 3)
		pathstart == ["api", "v0", "measurements"]
	
		token.payload.azp == "diwise-frontend"
	
		response := {
			"tenants": token.payload.tenants
		}
	}
	
	is_valid_token {
		1 == 1
	}
	
	token := {"payload": payload} {
		[_, payload, _] := io.jwt.decode(input.token)
	}
	`
const schneiderDataPointId = `
[
    {
        "name": "/MQTT-klient/!UC_Testvägen 17C/UC_Testvägen_17C_VS2_EM01-POWER/Value",
        "value": "11000",
        "unit": "W",
        "description": "Momentaneffekt VS2",
        "pointID": "nspg:xxyyzz.7ofmFSyvPEvCxiyquQ/Value"
    },
    {
        "name": "/Enterprise Server Mitthem/IoT-gränssnitt/MQTT-klient/!UC_Testvägen/UC_Testvägen_EM01-ENERGY/Value",
        "value": "448000000",
        "unit": "Wh",
        "description": "Mätarställning VS2",
        "pointID": "nspg:xxyyzz.puNUCJWo0gB3ycKMNQ/Value"
    },
    {
        "name": "/MQTT-klient/!LB02 Testvägen 17C/LB02_Testvägen_17C_SV21/Value",
        "value": "38.159999847412109",
        "unit": "%",
        "description": "Styrsignal värmeventil batteri",
        "pointID": "nspg:xxyyzz.uNjNSgzbEeLWpQ/Value"
    },
    {
        "name": "/MQTT-klient/!LB02 Testvägen 17C/LB02_Testvägen_17C_GP12_BB/Value",
        "value": "220",
        "unit": "Pa",
        "description": "Börvärde tryck frånluft",
        "pointID": "nspg:xxyyzz.eYytQMqHt/0eJ227IQ/Value"
    },
    {
        "name": "/MQTT-klient/!LB02 Testvägen 17C/LB02_Testvägen_17C_GF12/Value",
        "value": "626.46917724609375",
        "unit": "l/s",
        "description": "Flöde frånluft",
        "pointID": "nspg:xxyyzz.oV7vQTCB8oVvXiUe5w/Value"
    },
	{
		"name":"/MQTT-klient/!UC_Test/UC_TEST_VP1_EM01-T2/Value",
		"value":"41",
		"unit":"°C",
		"description":"Returtemperatur Värme Primär",
		"pointID":"nspg:xxyyzz.3TKO9xncT5Q8F9w/Value"
	}
]`
