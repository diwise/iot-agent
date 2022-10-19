package decoder

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
)

func SenlabTBasicDecoder(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {

	var p payload

	// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2)
	// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2) | Temp(2)
	if len(ue.Data) < 4 {
		return errors.New("payload too short")
	}

	err := decodePayload(ue.Data, &p)
	if err != nil {
		return err
	}

	temp := struct {
		Temperature float32 `json:"temperature"`
	}{
		p.Temperature,
	}

	bat := struct {
		BatteryLevel int `json:"battery_level"`
	}{
		p.BatteryLevel,
	}

	pp := &Payload{
		DevEUI:       ue.DevEui,
		Timestamp:    ue.Timestamp.Format(time.RFC3339Nano),
		BatteryLevel: bat.BatteryLevel,
	}
	pp.Measurements = append(pp.Measurements, temp)
	pp.Measurements = append(pp.Measurements, bat)

	err = fn(ctx, *pp)
	if err != nil {
		return err
	}

	return nil
}

type payload struct {
	ID           int
	BatteryLevel int
	Temperature  float32
}

func decodePayload(b []byte, p *payload) error {
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

func singleProbe(b []byte, p *payload) error {
	var temp int16
	err := binary.Read(bytes.NewReader(b[len(b)-2:]), binary.BigEndian, &temp)
	if err != nil {
		return err
	}

	p.ID = int(b[0])
	p.BatteryLevel = (int(b[1]) * 100) / 254
	p.Temperature = float32(temp) / 16.0

	return nil
}

func dualProbe(b []byte, p *payload) error {
	return errors.New("unsupported dual probe payload")
}
