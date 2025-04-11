package airquality

import (
	"context"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

func TestAirQuality(t *testing.T) {
	is := is.New(t)
	ue, err := facades.New("servanet")([]byte(airqualityData))
	is.NoErr(err)

	payload, err := Decoder(context.Background(), ue)
	is.NoErr(err)

	objects, err := Converter(context.Background(), "id", payload, ue.Timestamp)
	is.NoErr(err)

	is.Equal(objects[0].ID(), "id")

	hum := objects[0].(lwm2m.Humidity)
	temp := objects[1].(lwm2m.Temperature)
	air := objects[2].(lwm2m.AirQuality)

	is.Equal(hum.SensorValue, 76.8)
	is.Equal(temp.SensorValue, 1.7)
	is.Equal(*air.PM10, 10.170945308453014)
	is.Equal(*air.PM25, 8.457361482209397)
	is.Equal(*air.NO2, 2.1099999999999954)

	airPack := lwm2m.ToPack(air)
	rec, ok := airPack.GetRecord(senml.FindByName("0"))
	is.True(ok)
	is.Equal(rec.Name, "id/3428/0")
	is.Equal(rec.StringValue, "urn:oma:lwm2m:ext:3428")
}

const airqualityData string = `
{
  "data": "FgARAL0KAAMAAB0n",
  "fPort": 2,
  "devEui": "xxxxxxxxxxxxxxxx",
  "object": {
    "no2": 2.1099999999999954,
    "pm10": 10.170945308453014,
    "pm25": 8.457361482209397,
    "pm10raw": 2.2,
    "pm25raw": 1.7,
    "humidity": 76.8,
    "sensorerror": 0,
    "temperature": 1.7
  },
  "rxInfo": [{
    "snr": -5,
    "rssi": -112,
    "uplinkId": "9161393f-deff-4b4e-a211-60a722d652e7",
    "gatewayId": "xxxxxxxxxxxxxxxx"
  }],
  "txInfo": {},
  "timestamp": "2025-02-13T00:25:49.864785Z",
  "deviceName": "AirQuality_4",
  "sensorType": "AirQuality"
}`
