package senlabt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/lwm2m"
)

type SenlabPayload struct {
	ID           int
	BatteryLevel int
	Temperature  float64
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	var d SenlabPayload

	// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2)
	// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2) | Temp(2)
	if len(e.Data) < 4 {
		return nil, errors.New("payload too short")
	}

	err := decodePayload(e.Data, &d)
	if err != nil {
		return nil, err
	}

	objects := make([]lwm2m.Lwm2mObject, 0)

	objects = append(objects, lwm2m.Battery{
		ID_:          deviceID,
		BatteryLevel: d.BatteryLevel,
		Timestamp_:   e.Timestamp,
	})

	objects = append(objects, lwm2m.Temperature{
		ID_:         deviceID,
		Timestamp_:  e.Timestamp,
		SensorValue: lwm2m.Round(d.Temperature),
	})

	return objects, nil
}

func decodePayload(b []byte, p *SenlabPayload) error {
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

func singleProbe(b []byte, p *SenlabPayload) error {
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

func dualProbe(b []byte, p *SenlabPayload) error {
	return errors.New("unsupported dual probe payload")
}
