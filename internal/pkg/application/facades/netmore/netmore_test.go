package netmore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/matryer/is"
)

func TestUplink(t *testing.T) {
	is := is.New(t)
	var uplinkEvents []UplinkEvent
	err := json.Unmarshal([]byte(up), &uplinkEvents)
	is.NoErr(err)
}

func TestUplinkAll(t *testing.T) {
	is := is.New(t)
	var uplinkEvents []UplinkEvent
	err := json.Unmarshal([]byte(upall), &uplinkEvents)
	is.NoErr(err)
}

func TestHandleUplinkAll(t *testing.T) {
	is := is.New(t)
	ue, err := HandleEvent(context.Background(), "payload", []byte(upall))
	is.NoErr(err)
	is.Equal(ue.DevEUI, "363536305d398e11")
}

func TestHandleUplink(t *testing.T) {
	is := is.New(t)
	ue, err := HandleEvent(context.Background(), "payload", []byte(up))
	is.NoErr(err)
	is.Equal(ue.DevEUI, "70b3d554600002e7")
}

const up string = `[{"devEui":"70b3d554600002e7","sensorType":"cube02","timestamp":"2025-04-10T20:48:22.053Z","payload":"b006b800013008cc98000002b8a8000399000000190840e40000","spreadingFactor":"12","dr":0,"rssi":"-104","snr":"-2","gatewayIdentifier":"824","messageType":"payload","fPort":"2"}]`
const upall string = `[{"devEui":"363536305d398e11","sensorType":"other","timestamp":"2025-04-10T15:22:58.961Z","payload":"809836","freq":867300000,"spreadingFactor":"12","dr":0,"rssi":"-115","snr":"-8.2","gatewayIdentifier":"1789","tags":{"GBG_KOV":[]},"gateways":[{"rssi":"-125","snr":"-17.2","antenna":0,"gatewayIdentifier":"19198","gwEui":"7076ff0056081cc9","mac":"7076ff03a7b0"},{"rssi":"-115","snr":"-8.2","antenna":0,"gatewayIdentifier":"1789","gwEui":"647fdafffe016c40","mac":"647fda016c40"},{"rssi":"-113","snr":"-17","antenna":0,"gatewayIdentifier":"187","gwEui":"00800000a0001f6d","mac":"0008004a35ae"}],"messageType":"payload","fCntUp":4427,"batteryLevel":"0","ack":false,"fPort":"2"}]`
