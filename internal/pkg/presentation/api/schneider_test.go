package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSchneiderHandler(t *testing.T) {
	is, api, app := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	api.forwardingEndpoint = server.URL + "/api/v0/messages"

	resp, _ := testRequest(is, http.MethodPost, api.forwardingEndpoint+"/schneider", bytes.NewBuffer([]byte(schneiderDataPointId)))
	is.Equal(resp.StatusCode, http.StatusCreated)            // status code should be 201
	is.Equal(len(app.HandleSensorMeasurementListCalls()), 4) // should be 4 - once for each supported object in schneider data
}

func TestSchneiderHandler2(t *testing.T) {
	is, api, app := testSetup(t)

	server := httptest.NewServer(api.r)
	defer server.Close()

	api.forwardingEndpoint = server.URL + "/api/v0/messages"

	resp, _ := testRequest(is, http.MethodPost, api.forwardingEndpoint+"/schneider", bytes.NewBuffer([]byte(payload)))
	is.Equal(resp.StatusCode, http.StatusCreated)            // status code should be 201
	is.Equal(len(app.HandleSensorMeasurementListCalls()), 14) // should be 14 - once for each supported object in schneider data
}

const payload string = `[
  {
    "name": "/iot/!ucf/ucf_VP1_EM01-T2/Value",
    "value": "14",
    "unit": "°C",
    "description": "Returtemperatur Värme Primär",
    "pointID": "nspg:abc.3TKO9xncTF+mnfOC5Q8F9w//Value"
  },
  {
    "name": "/iot/!ucf/ucf_VP1_EM01-T1/Value",
    "value": "90",
    "unit": "°C",
    "description": "Tilloppstemperatur Värme Primär",
    "pointID": "nspg:abc.2UqZEOCHSg+pSqn8xs4pKA//Value"
  },
  {
    "name": "/iot/!ucv/ucv_VV1_EM01-POWER/Value",
    "value": "0",
    "unit": "W",
    "description": "Momentaneffekt VV1",
    "pointID": "nspg:abc.qUDse9FTQ6SAlCnTvkssNg//Value"
  },
  {
    "name": "/iot/!ucv/ucv_VV1_EM01-ENERGY/Value",
    "value": "1689000000",
    "unit": "Wh",
    "description": "Mätarställning VV1",
    "pointID": "nspg:abc.MFtjO3FeTTqQ2oSXC7SyBg//Value"
  },
  {
    "name": "/iot/!ucv/ucv_VP1_EM01-POWER/Value",
    "value": "0",
    "unit": "W",
    "description": "Momentaneffekt VP1",
    "pointID": "nspg:abc.Bg2TMm3sQ9ejFHPjAyT9uQ//Value"
  },
  {
    "name": "/iot/!ucv/ucv_VP1_EM01-ENERGY/Value",
    "value": "536000000",
    "unit": "Wh",
    "description": "Mätarställning VP1",
    "pointID": "nspg:abc.vindR9U+Qdqt/Gr7smlqkQ//Value"
  },
  {
    "name": "/iot/!ucn 17C/ucn_17C_VS1_EM01-ENERGY/Value",
    "value": "1689000000",
    "unit": "Wh",
    "description": "Mätarställning VS1",
    "pointID": "nspg:abc.tJsAXXsEQbu1C2ryFwpkVw//Value"
  },
  {
    "name": "/iot/!ucn 17C/ucn_17C_VS1_EM01-POWER/Value",
    "value": "0",
    "unit": "W",
    "description": "Momentaneffekt VS1",
    "pointID": "nspg:abc.p6NPxxVeSByFQuAcTqZnDQ//Value"
  },
  {
    "name": "/iot/!ucn 17C/ucn_17C_VS2_EM01-ENERGY/Value",
    "value": "536000000",
    "unit": "Wh",
    "description": "Mätarställning VS2",
    "pointID": "nspg:abc.puNUCJGCQXWo0gB3ycKMNQ//Value"
  },
  {
    "name": "/iot/!ucn 17C/ucn_17C_VS2_EM01-POWER/Value",
    "value": "0",
    "unit": "W",
    "description": "Momentaneffekt VS2",
    "pointID": "nspg:abc.7ofmFSylTjWvPEvCxiyquQ//Value"
  },
  {
    "name": "/iot/!ucf/ucf_VV1_EM01-ENERGY/Value",
    "value": "215000000",
    "unit": "Wh",
    "description": "Mätarställning Varmvatten",
    "pointID": "nspg:abc.hlEih+U/Sp6iH1UIdx/eGQ//Value"
  },
  {
    "name": "/iot/!ucf/ucf_VV1_EM01-POWER/Value",
    "value": "10000",
    "unit": "W",
    "description": "Momentaneffekt Varmvatten",
    "pointID": "nspg:abc.gyQh06n/QzOyIyyvghT3uQ//Value"
  },
  {
    "name": "/iot/!ucf/ucf_VP1_EM01-ENERGY/Value",
    "value": "3829000000",
    "unit": "Wh",
    "description": "Mätarställning Värme Primär",
    "pointID": "nspg:abc.2764r8xFTx+/dbrKTAhZww//Value"
  },
  {
    "name": "/iot/!ucf/ucf_VP1_EM01-POWER/Value",
    "value": "50000",
    "unit": "W",
    "description": "Momentaneffekt Värme Primär",
    "pointID": "nspg:abc.R6oHmMvkSjWgckqLxPAyNA//Value"
  }
]`
