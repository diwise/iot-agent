package talkpool

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

var (
	ErrUnsupportedPort = errors.New("oy1210: unsupported fPort, expected 2")
	ErrInvalidLength   = errors.New("oy1210: payload must contain exactly 5 bytes")
)

type Oy1210Data struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	CO2         int     `json:"co2"`
}

func (a Oy1210Data) BatteryLevel() *int {
	return nil
}
func (a Oy1210Data) Error() (string, []string) {
	return "", []string{}
}

func DecoderOy1210(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	p, err := decodeOy1210Payload(e.Payload.Data, 2)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func decodeOy1210Payload(bytes []byte, port int) (*Oy1210Data, error) {
	if port != 2 {
		return nil, ErrUnsupportedPort
	}

	if len(bytes) != 5 {
		return nil, ErrInvalidLength
	}

	tempRaw := (int(bytes[0]) << 4) | (int(bytes[2]) >> 4)
	humidityRaw := (int(bytes[1]) << 4) | int(bytes[2])&0x0F
	co2Raw := (int(bytes[3]) << 8) | int(bytes[4])

	return &Oy1210Data{
		Temperature: roundToOneDecimal(float64(tempRaw)/10.0 - 80.0),
		Humidity:    roundToOneDecimal(float64(humidityRaw)/10.0 - 25.0),
		CO2:         co2Raw,
	}, nil
}

func roundToOneDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}

func ConverterOy1210(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(Oy1210Data)
	return convertToLwm2mObjects(ctx, deviceID, p, ts, "sht3x"), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p Oy1210Data, ts time.Time, options ...string) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	objects = append(objects, lwm2m.NewTemperature(deviceID, float64(p.Temperature), ts))

	objects = append(objects, lwm2m.NewHumidity(deviceID, float64(p.Humidity), ts))

	co2 := float64(p.CO2)
	objects = append(objects, lwm2m.NewAirQuality(deviceID, &co2, nil, nil, nil, ts))

	return objects
}
