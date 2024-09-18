package milesight

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

func TestMilesightAM100Decoder(t *testing.T) {
	is, _ := testSetup(t)
	ue, _ := application.ChirpStack([]byte(data_am100))

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	b, _ := objects[0].(lwm2m.Device)
	is.Equal(*b.BatteryLevel, int(89))

	co2, _ := objects[1].(lwm2m.AirQuality)
	is.Equal(*co2.CO2, float64(886))

	h, _ := objects[2].(lwm2m.Humidity)
	is.Equal(h.SensorValue, float64(29))

	tmp, _ := objects[3].(lwm2m.Temperature)
	is.Equal(tmp.SensorValue, float64(22.3))
}

func TestMilesightEM500Decoder(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.ChirpStack([]byte(data_em500))

	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	d, _ := objects[0].(lwm2m.Distance)
	is.Equal(d.SensorValue, float64(5.0))
}

func TestMilesightDecoderEM400TLD(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.ChirpStack([]byte(data_em400))
	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	device, _ := objects[0].(lwm2m.Device)
	is.Equal(*device.BatteryLevel, 98)
	distance, _ := objects[1].(lwm2m.Distance)
	is.Equal(distance.SensorValue, float64(0.267))
}

func TestMilesightDecoderEM400TLD2(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.ChirpStack([]byte(data_em400_tld))
	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	device, _ := objects[0].(lwm2m.Device)
	is.Equal(*device.BatteryLevel, 100)
	distance, _ := objects[1].(lwm2m.Distance)
	is.Equal(distance.SensorValue, float64(0.818))
	temerature, _ := objects[2].(lwm2m.Temperature)
	is.Equal(9.8, temerature.SensorValue)
}

func TestMilesightDecoderEM400TLD_NegTemp(t *testing.T) {
	is, _ := testSetup(t)

	ue, _ := application.ChirpStack([]byte(data_em400_tld_neg))
	objects, err := Decoder(context.Background(), "devid", ue)
	is.NoErr(err)
	is.Equal(objects[0].ID(), "devid")

	device, _ := objects[0].(lwm2m.Device)
	is.Equal(*device.BatteryLevel, 100)
	distance, _ := objects[1].(lwm2m.Distance)
	is.Equal(distance.SensorValue, float64(0.710))
	temerature, _ := objects[2].(lwm2m.Temperature)
	is.Equal(-0.8, temerature.SensorValue)
}

func TestMilesightEM300Decoder(t *testing.T) {
	is := is.New(t)
	ue, _ := application.ChirpStack([]byte(data_em300))
	m := milesightdecoder(ue.Data)
	is.True(m != nil)
}

func TestMilesightEM300DecoderLwm2m(t *testing.T) {
	is := is.New(t)
	ue, _ := application.ChirpStack([]byte(data_em300))
	m, err := Decoder(context.Background(), "device_id", ue)
	is.NoErr(err)

	is.True(!m[2].(lwm2m.DigitalInput).DigitalInputState)
}

func TestV4(t *testing.T) {
	is := is.New(t)
	ue, _ := application.ChirpStack([]byte(data_em400_tld_neg))
	m := milesightdecoder(ue.Data)
	is.True(m != nil)
}

func testSetup(t *testing.T) (*is.I, *slog.Logger) {
	is := is.New(t)
	return is, slog.New(slog.NewTextHandler(io.Discard, nil))
}

const data_am100 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"AM103_1",
	"deviceProfileName":"Milesight AM100",
	"deviceProfileID":"c6a3467d-519d-4861-8e90-ba13a7b7c9ee",
	"devEUI":"24e124725c140542",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"AXVZA2ffAARoOgd9dgM=",
	"object":
	{
		"battery":89,
		"co2":886,
		"humidity":29,
		"temperature":22.3
	}
}`

const data_em500 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"EM500_UDL_1",
	"deviceProfileName":"Milesight EM500",
	"deviceProfileID":"f865a295-3d90-424e-967c-133c35d5594c",
	"devEUI":"24e124126d154397",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"A4KIEw==",
	"object":
	{
		"distance":5000
	}
}`

const data_em400 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"EM400_TLD",
	"deviceProfileName":"Milesight EM400",
	"deviceProfileID":"f865a295-3d90-424e-967c-133c35d5594c",
	"devEUI":"24e124126d154397",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"AXViA2f9/wSCCwEFAAA=",
	"object":
	{
		"battery": 98,
		"distance": 267,
		"position": "normal",
		"temperature": -0.3
	}
}`

const data_em300 string = `{
	"applicationID":"71",
	"applicationName":"ncksalnckls",
	"deviceName":"EM400_TLD",
	"deviceProfileName":"Milesight EM400",
	"deviceProfileID":"f865a295-3d90-424e-967c-133c35d5594c",
	"devEUI":"24e124126d154397",
	"txInfo":
	{
		"frequency":868100000,
		"dr":5
	},
	"adr":true,
	"fCnt":10901,
	"fPort":5,
	"data":"A2fzAARocQYAAA==",
	"object":
	{
		"battery": 98,
		"distance": 267,
		"position": "normal",
		"temperature": -0.3
	}
}`



const data_em400_tld = `{"applicationID":"102","applicationName":"IoT","deviceName":"345","deviceProfileName":"Milesight EM400TLD","deviceProfileID":"c70ad8ac58","devEUI":"24reteyrty8855","rxInfo":[{"gatewayID":"rty","uplinkID":"9767bbed","name":"001","rssi":-111,"loRaSNR":-4.2,"location":{"latitude":62.0,"longitude":17.7,"altitude":0}},{"gatewayID":"tyu","uplinkID":"beaa8e","name":"S47","time":"2024-03-19T12:15:41.681927Z","rssi":-112,"loRaSNR":-0.5,"location":{"latitude":62.0,"longitude":17.9,"altitude":9}}],"txInfo":{"frequency":867900000,"dr":5},"adr":true,"fCnt":263,"fPort":85,"data":"AXVkA2diAASCMgMFAAA=","object":{"battery":100,"distance":818,"position":"normal","temperature":9.8},"tags":{"latitude":"62.9","longitude":"17.9","soptunneid":"xyz","typ":"160"}}`

const data_em400_tld_neg = `{"applicationID":"102","applicationName":"IoT","deviceName":"24","deviceProfileName":"Milesight EM400TLD","deviceProfileID":"c7058","devEUI":"24reteyrty8855","rxInfo":[{"gatewayID":"","uplinkID":"622cf7a0----","name":"SN--","time":"2024-03-25T09:24:18.179737579Z","rssi":-114,"loRaSNR":1.5,"location":{"latitude":62,"longitude":17,"altitude":7}}],"txInfo":{"frequency":868100000,"dr":4},"adr":true,"fCnt":100,"fPort":85,"data":"AXVkA2f4/wSCxgIFAAA=","object":{"battery":100,"distance":710,"position":"normal","temperature":-0.8}}`
