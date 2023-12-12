package sensefarm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type SensefarmPayload struct {
	TransmissionReason int8    // 0 = unknown reset, 1 = POR/PDR reset, 2 = Independt watchdog reset, 3 = windows watchdog reset, 4 = low power reset, 5 = POR/PDR reset, 6 = Normal transmission, 7 = Button reset
	ProtocolVersion    int16   // Version 0 -> 65535
	BatteryVoltage     int16   // 0 -> 65535 mV
	Resistances        []int32 // 0 -> 4294967295 Ohm
	SoilMoistures      []int16 // 0 -> 65535 kPa. If value is too low or too high (sensor not placed outdoors, cable broke, etc), the sensor data is considered invalid and may not be sent./
	Temperature        float32 // °C
}

func Decoder(ctx context.Context, deviceID string, e application.SensorEvent) ([]lwm2m.Lwm2mObject, error) {
	var psf SensefarmPayload

	// At minimum we must receive 2 bytes, one for header type and one for value
	if len(e.Data) < 2 {
		return nil, errors.New("payload too short")
	}

	psf, err := decodeSensefarmPayload(e.Data)
	if err != nil {
		return nil, err
	}

	objects := make([]lwm2m.Lwm2mObject, 0)

	objects = append(objects, lwm2m.Temperature{
		ID_:         deviceID,
		Timestamp_:  e.Timestamp,
		SensorValue: lwm2m.Round(float64(psf.Temperature)),
	})

	v := float64(psf.BatteryVoltage)
	objects = append(objects, lwm2m.Battery{
		ID_:            deviceID,
		Timestamp_:     e.Timestamp,
		BatteryVoltage: &v,
	})

	for _, r := range psf.Resistances {
		objects = append(objects, lwm2m.Conductivity{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: 1 / float64(r),
		})
	}

	for _, sm := range psf.SoilMoistures {
		objects = append(objects, lwm2m.Pressure{
			ID_:         deviceID,
			Timestamp_:  e.Timestamp,
			SensorValue: float64(sm * 1000),
		})
	}

	return objects, nil
}

func decodeSensefarmPayload(b []byte) (SensefarmPayload, error) {
	p := SensefarmPayload{}

	if len(b) == 0 {
		return p, errors.New("input payload array is empty")
	}

	for i := 0; i < len(b); i++ { //The multisensor message are read byte by byte and parsed for information on each individual sensor and it's values.
		switch (b[i] & 0xFF) >> 3 {
		case 0x01: //  Temperature
			noOfBytes := 4
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.Temperature)
			if err != nil {
				return p, fmt.Errorf("failed to read temperature: %w", err)
			}
			i += noOfBytes

		case 0x06: // Battery
			noOfBytes := 2
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.BatteryVoltage)
			if err != nil {
				return p, fmt.Errorf("failed to read battery: %w", err)
			}

			i += noOfBytes

		case 0x13: //Resistance
			var resistance int32

			noOfBytes := 4
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &resistance)

			if err != nil {
				return p, fmt.Errorf("failed to read resistance: %w", err)
			}

			p.Resistances = append(p.Resistances, resistance)
			i += noOfBytes

		case 0x15: // Soil moisture
			var soilMoisture int16

			noOfBytes := 2
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &soilMoisture)
			if err != nil {
				return p, fmt.Errorf("failed to read soil moisture: %w", err)
			}

			p.SoilMoistures = append(p.SoilMoistures, soilMoisture)
			i += noOfBytes

		case 0x016: // Transmission reason
			noOfBytes := 1
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.TransmissionReason)
			if err != nil {
				return p, fmt.Errorf("failed to read transmission reason: %w", err)
			}

			i += noOfBytes

		case 0x17: // Protocol version
			noOfBytes := 2
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.ProtocolVersion)
			if err != nil {
				return p, fmt.Errorf("failed to read protocol version: %w", err)
			}

			i += noOfBytes

		default:
		}
	}

	return p, nil
}
