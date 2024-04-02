package vegapuls

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type VegapulsPayload struct {
	Battery     *int
	Distance    *float64
	Temperature *float64
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	p, err := decode(e.Data)
	if err != nil {
		return nil, err
	}

	return convertToLwm2mObjects(ctx, deviceID, p, e.Timestamp), nil
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
		objects = append(objects, lwm2m.NewDistance(deviceID, *p.Distance, ts))
	}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, *p.Temperature, ts))
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

func decode(b []byte) (VegapulsPayload, error) {
	p := VegapulsPayload{}

	numberOfBytes := len(b)

	fmt.Printf("%d\n", numberOfBytes)

	if numberOfBytes < 11 {
		return p, fmt.Errorf("incomplete or partial payload")
	}

	if int(b[0]) != 2 {
		return p, fmt.Errorf("unknown packet indicator")
	}

	distBits := binary.BigEndian.Uint32(b[2:6])
	distance := math.Float64frombits(uint64(distBits))
	p.Distance = &distance

	battery := int(b[7])
	p.Battery = &battery

	tempBits := binary.BigEndian.Uint16(b[8:10])
	temp := float64(tempBits) / 10
	p.Temperature = &temp

	return p, nil
}
