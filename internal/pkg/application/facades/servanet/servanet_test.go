package servanet

import (
	"encoding/json"
	"testing"

	"github.com/matryer/is"
)

func TestUplinkEvent(t *testing.T) {
	is := is.New(t)
	var uplinkEvent UplinkEvent
	err := json.Unmarshal([]byte(up), &uplinkEvent)
	is.NoErr(err)
}

func TestErrorEvent(t *testing.T) {
	is := is.New(t)
	var errorEvent ErrorEvent
	err := json.Unmarshal([]byte(err), &errorEvent)
	is.NoErr(err)
}

func TestStatusEvent(t *testing.T) {
	is := is.New(t)
	var statusEvent StatusEvent
	err := json.Unmarshal([]byte(status), &statusEvent)
	is.NoErr(err)
}

func TestHandleUplinkEvent(t *testing.T) {
	is := is.New(t)
	ue, err := HandleUplinkEvent([]byte(up))
	is.NoErr(err)
	is.Equal(ue.DevEui, "24e124329e090021")
}

const up string = `{"applicationID":"102","applicationName":"3_IoT-For-Klimat","deviceName":"TLD-01","deviceProfileName":"Milesight EM400TLD","deviceProfileID":"c70ad992-b55e-4f40-804b-2ebfec18ac58","devEUI":"24e124329e090021","rxInfo":[{"gatewayID":"24e124fffef477f6","uplinkID":"68bae122-70fd-4487-866d-49ccf45e9ab4","name":"SN-LGW-047","time":"2025-04-10T11:44:01.912259Z","rssi":-110,"loRaSNR":-5.8,"location":{"latitude":62.36951,"longitude":17.32014,"altitude":273}},{"gatewayID":"fcc23dfffe0a752b","uplinkID":"058471f1-3076-47a3-9ef1-6d1ad5bd248f","name":"SN-LGW-001","rssi":-104,"loRaSNR":1.2,"location":{"latitude":62.39466886148298,"longitude":17.34076023101807,"altitude":0}}],"txInfo":{"frequency":868500000,"dr":5},"adr":true,"fCnt":45797,"fPort":85,"data":"AXVXA2c4AASCXAgFAAA=","object":{"battery":87,"distance":2140,"position":"normal","temperature":5.6},"tags":{"x_typ":"mr_hushall"}}`
const err string = `{"applicationID":"102","applicationName":"3_IoT-For-Klimat","deviceName":"TLD-01","devEUI":"24e124329e090021","type":"UPLINK_FCNT_RETRANSMISSION","error":"frame-counter did not increment","fCnt":45797,"tags":{"x_typ":"mr_hushall"}}`
const status string = `{"applicationID":"2","applicationName":"1_Watermetering","deviceName":"07624101","devEUI":"8c83fc05007455a5","margin":29,"externalPowerSource":false,"batteryLevel":95.67,"batteryLevelUnavailable":false}`
