package vegapuls

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type VegapulsPayload struct {
	Battery     *int
	Distance    *float32
	Temperature *float64
	Unit        *uint32
}

func (a VegapulsPayload) BatteryLevel() *int {
	return a.Battery
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	return decode(e.Payload.Data)
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(VegapulsPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p VegapulsPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Battery != nil {
		d := lwm2m.NewDevice(deviceID, ts)
		bat := int(*p.Battery)
		d.BatteryLevel = &bat
		objects = append(objects, d)
	}

	if p.Distance != nil {
		dist := roundFloat(float64(*p.Distance), 5)
		objects = append(objects, lwm2m.NewDistance(deviceID, dist, ts))
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, *p.Temperature, ts))
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

var packetLengthByIndicator map[int]int = map[int]int{
	2:  11,
	8:  11,
	12: 20,
}

var distanceUnit map[uint32]string = map[uint32]string{
	44: "ft",
	45: "m",
	47: "inch",
	49: "mm",
}

const fahrenheit uint32 = 33

func decode(b []byte) (VegapulsPayload, error) {
	p := VegapulsPayload{}

	packetIndicator := int(b[0])

	packetLength, ok := packetLengthByIndicator[packetIndicator]
	if !ok {
		return p, fmt.Errorf("unknown packet indicator")
	}

	if len(b) < packetLength {
		return p, fmt.Errorf("incomplete or partial payload")
	}

	pos := 0

	if packetIndicator != 2 {
		pos = -1
	}

	distBits := binary.BigEndian.Uint32(b[pos+2 : pos+6])
	distance := math.Float32frombits(distBits)

	// convert from not meters to meters
	switch {
	case distanceUnit[uint32(b[pos+6])] == "ft":
		distance = distance * 0.3048
	case distanceUnit[uint32(b[pos+6])] == "inch":
		distance = distance * 0.0254
	case distanceUnit[uint32(b[pos+6])] == "mm":
		distance = distance * 1000
	}

	p.Distance = &distance

	if packetIndicator == 2 {
		packetLength = packetLength + 1
	}

	battery := int(b[packetLength-5])
	p.Battery = &battery

	tempBits := binary.BigEndian.Uint16(b[packetLength-4 : packetLength-2])
	temp := float64(tempBits) / 10

	if uint32(b[packetLength-2]) == fahrenheit {
		f := (temp - 32) * 5 / 9
		temp = f
	}

	p.Temperature = &temp

	return p, nil
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
