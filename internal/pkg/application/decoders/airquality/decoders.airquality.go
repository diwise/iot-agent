package airquality

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type AirQualityPayload struct {
	PM10Raw     float64  `json:"pm10raw"`
	PM10        *float64 `json:"pm10"`
	PM25Raw     float64  `json:"pm25raw"`
	PM25        *float64 `json:"pm25"`
	Temperature *float64 `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
	SensorError uint16   `json:"sensorerror"`
	NO2         *float64 `json:"no2"`
}

func Decoder(ctx context.Context, e types.SensorEvent) (any, error) {

	if e.FPort != 2 {
		return nil, errors.New("invalid fPort")
	}

	return decode(e.Data)
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(AirQualityPayload)
	return convertToLwm2mObjects(deviceID, p, ts), nil
}

func convertToLwm2mObjects(deviceID string, p AirQualityPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Humidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, *p.Humidity, ts))
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, *p.Temperature, ts))
	}

	aq := lwm2m.NewAirQuality(deviceID, nil, p.PM10, p.PM25, p.NO2, ts)

	objects = append(objects, aq)

	return objects
}

func decode(bytes []byte) (AirQualityPayload, error) {
	var aqp AirQualityPayload

	if len(bytes) < 12 {
		err := fmt.Errorf("not enough bytes to decode")
		return aqp, err
	}

	aqp.PM10Raw = float64(uint16(bytes[1])<<8|uint16(bytes[0])) / 10.0
	pm10 := math.Max(math.Log(aqp.PM10Raw)*6.6462+4.9307, 0)
	aqp.PM10 = &pm10

	aqp.PM25Raw = float64(uint16(bytes[3])<<8|uint16(bytes[2])) / 10.0
	pm25 := math.Max(math.Log(aqp.PM25Raw)*6.6462+4.9307, 0)
	aqp.PM25 = &pm25

	temp := float64((uint16(bytes[5])<<8|uint16(bytes[4]))-2732) / 10.0
	aqp.Temperature = &temp

	humidity := float64(uint16(bytes[7])<<8|uint16(bytes[6])) / 10.0
	aqp.Humidity = &humidity

	aqp.SensorError = uint16(bytes[9])<<8 | uint16(bytes[8])

	no2Value := float64(uint16(bytes[11])<<8|uint16(bytes[10]))/100.0 - 100.0 + 1.98
	no2 := math.Max(no2Value, 0)
	aqp.NO2 = &no2

	return aqp, nil
}
