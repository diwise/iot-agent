package niab

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func Decoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {

	if len(ue.Data) < 4 {
		return errors.New("payload too short")
	} else if len(ue.Data) > 4 {
		return errors.New("payload too long")
	}

	decorators, err := decodePayload(ue.Data)
	if err != nil {
		return err
	}

	p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...)
	if err != nil {
		return err
	}

	return fn(ctx, p)
}

func decodePayload(b []byte) ([]payload.PayloadDecoratorFunc, error) {

	battery := int(b[0])
	temp := int(b[1])

	if temp > 127 {
		temp = temp - 255
	}

	decorators := append(
		make([]payload.PayloadDecoratorFunc, 0, 3),
		payload.BatteryLevel(battery*100/255),
		payload.Temperature(float64(temp)),
	)

	var distance int16
	binary.Read(bytes.NewReader(b[2:4]), binary.BigEndian, &distance)

	// filter out sensor reading errors
	// TODO: Figure out if we want to signal this error in some way
	if distance != -1 {
		// convert the reported distance in millimeters to meters instead
		decorators = append(decorators, payload.Distance(float64(distance)/1000.0))
	}

	return decorators, nil
}
