package decoder

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

type payloadSensefarm struct {
	TransmissionReason int8    // 0 = unknown reset, 1 = POR/PDR reset, 2 = Independt watchdog reset, 3 = windows watchdog reset, 4 = low power reset, 5 = POR/PDR reset, 6 = Normal transmission, 7 = Button reset
	ProtocolVersion    int16   // Version 0 -> 65535
	BatteryVoltage     int16   // 0 -> 65535 mV
	Resistances        []int32 // 0 -> 4294967295 Ohm
	SoilMoistures      []int16 // 0 -> 65535 kPa. If value is too low or too high (sensor not placed outdoors, cable broke, etc), the sensor data is considered invalid and may not be sent./
	Temperature        float32 // °C
}

func SensefarmBasicDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	var psf payloadSensefarm

	// At minimum we must receive 2 bytes, one for header type and one for value
	if len(ue.Data) < 2 {
		return errors.New("payload too short")
	}

	err := decodeSensefarmPayload(ue.Data, &psf)
	if err != nil {
		return err
	}

	p, err := payload.New(ue.DevEui, ue.Timestamp,
		payload.TransmissionReason(psf.TransmissionReason),
		payload.ProtocolVersion(int8(psf.ProtocolVersion)),
		payload.BatteryVoltage(int(psf.BatteryVoltage)),
		payload.Resistance(psf.Resistances),
		payload.SoilMoisture(psf.SoilMoistures),
		payload.Temperature(psf.Temperature))
	if err != nil {
		return err
	}

	return fn(ctx, p)
}

func decodeSensefarmPayload(b []byte, p *payloadSensefarm) error {
	if len(b) == 0 {
		return fmt.Errorf("input payload array is empty")
	}

	for i := 0; i < len(b); i++ { //The multisensor message are read byte by byte and parsed for information on each individual sensor and it's values.
		switch (b[i] & 0xFF) >> 3 {
		case 0x01: //  Temperature
			noOfBytes := 4
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.Temperature)
			if err != nil {
				return fmt.Errorf("failed to read temperature: %w", err)
			}
			i += noOfBytes

		case 0x06: // Battery
			noOfBytes := 2
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.BatteryVoltage)
			if err != nil {
				return fmt.Errorf("failed to read battery: %w", err)
			}

			i += noOfBytes

		case 0x13: //Resistance
			var resistance int32

			noOfBytes := 4
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &resistance)

			if err != nil {
				return fmt.Errorf("failed to read resistance: %w", err)
			}

			p.Resistances = append(p.Resistances, resistance)
			i += noOfBytes

		case 0x15: // Soil moisture
			var soilMoisture int16

			noOfBytes := 2
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &soilMoisture)
			if err != nil {
				return fmt.Errorf("failed to read soil moisture: %w", err)
			}

			p.SoilMoistures = append(p.SoilMoistures, soilMoisture)
			i += noOfBytes

		case 0x016: // Transmission reason
			noOfBytes := 1
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.TransmissionReason)
			if err != nil {
				return fmt.Errorf("failed to read transmission reason: %w", err)
			}

			i += noOfBytes

		case 0x17: // Protocol version
			noOfBytes := 2
			err := binary.Read(bytes.NewReader(b[i+1:]), binary.BigEndian, &p.ProtocolVersion)
			if err != nil {
				return fmt.Errorf("failed to read protocol version: %w", err)
			}

			i += noOfBytes

		default:
		}
	}

	return nil
}
