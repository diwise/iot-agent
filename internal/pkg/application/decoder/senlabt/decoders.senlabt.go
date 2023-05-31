package senlabt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

type sensorData struct {
	ID           int
	BatteryLevel int
	Temperature  float64
}

func SenlabTBasicDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {

	var d sensorData

	// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2)
	// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2) | Temp(2)
	if len(ue.Data) < 4 {
		return errors.New("payload too short")
	}

	err := decodePayload(ue.Data, &d)
	if err != nil {
		return err
	}

	p, err := payload.New(ue.DevEui, ue.Timestamp, payload.BatteryLevel(d.BatteryLevel), payload.Temperature(d.Temperature))
	if err != nil {
		return err
	}

	return fn(ctx, p)
}

func decodePayload(b []byte, p *sensorData) error {
	id := int(b[0])
	if id == 1 {
		err := singleProbe(b, p)
		if err != nil {
			return err
		}
	}
	if id == 12 {
		err := dualProbe(b, p)
		if err != nil {
			return err
		}
	}

	// these values must be ignored since they are sensor reading errors
	if p.Temperature == -46.75 || p.Temperature == 85 {
		return errors.New("sensor reading error")
	}

	return nil
}

func singleProbe(b []byte, p *sensorData) error {
	var temp int16
	err := binary.Read(bytes.NewReader(b[len(b)-2:]), binary.BigEndian, &temp)
	if err != nil {
		return err
	}

	p.ID = int(b[0])
	p.BatteryLevel = (int(b[1]) * 100) / 254
	p.Temperature = float64(temp) / 16.0

	return nil
}

func dualProbe(b []byte, p *sensorData) error {
	return errors.New("unsupported dual probe payload")
}
