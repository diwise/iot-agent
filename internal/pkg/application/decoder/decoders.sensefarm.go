package decoder

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

func SensefarmBasicDecoder(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {

	dm := []struct {
		DevEUI            string  `json:"devEUI"`
		SensorType        string  `json:"sensorType,omitempty"`
		Timestamp         string  `json:"timestamp,omitempty"`
		Payload           string  `json:"payload"`
		SpreadingFactor   string  `json:"spreadingfactor,omitempty"`
		Rssi              string  `json:"rssi,omitempty"`
		Snr               string  `json:"snr,omitempty"`
		Latitude          float64 `json:"latitude,omitempty"`
		Longitude         float64 `json:"longitude,omitempty"`
		GatewayIdentifier string  `json:"gatewayIdentifier,omitempty"`
		FPort             string  `json:"fPort,omitempty"`
	}{}

	err := json.Unmarshal(msg, &dm)
	if err != nil {
		return err
	}

	var p payloadSensefarm
	for _, d := range dm {

		b, err := hex.DecodeString(d.Payload)
		if err != nil {
			return err
		}

		// At minimum we must receive 2 bytes, one for header type and one for value
		if len(b) < 2 {
			return errors.New("payload too short")
		}

		err = decodeSensefarmPayload(b, &p)
		if err != nil {
			return err
		}

		transmissionReason := struct {
			TransmissionReason uint `json:"transmission_reason"`
		}{
			uint(p.TransmissionReason),
		}

		protocolVersion := struct {
			ProtocolVersion uint `json:"protocol_version"`
		}{
			uint(p.ProtocolVersion),
		}

		batteryVoltage := struct {
			BatteryVoltage uint `json:"battery_voltage"`
		}{
			p.BatteryVoltage,
		}

		resistance := struct {
			Resistance []uint64 `json:"resistance"`
		}{
			p.Resistance,
		}

		soilMoisture := struct {
			SoilMoisture []uint32 `json:"soil_moisture"`
		}{
			p.SoilMoisture,
		}

		temperature := struct {
			Temperature float32 `json:"temperature"`
		}{
			p.Temperature,
		}

		pp := &Payload{
			DevEUI:            d.DevEUI,
			FPort:             d.FPort,
			SpreadingFactor:   d.SpreadingFactor,
			Rssi:              d.Rssi,
			Snr:               d.Snr,
			Latitude:          d.Latitude,
			Longitude:         d.Longitude,
			GatewayIdentifier: d.GatewayIdentifier,
			SensorType:        d.SensorType,
			Timestamp:         d.Timestamp,
		}

		pp.Measurements = append(pp.Measurements, transmissionReason)
		pp.Measurements = append(pp.Measurements, protocolVersion)
		pp.Measurements = append(pp.Measurements, batteryVoltage)
		pp.Measurements = append(pp.Measurements, resistance)
		pp.Measurements = append(pp.Measurements, soilMoisture)
		pp.Measurements = append(pp.Measurements, temperature)

		err = fn(ctx, *pp)
		if err != nil {
			return err
		}
	}

	return nil
}

type payloadSensefarm struct {
	TransmissionReason uint     // 	0 = unknown reset, 1 = POR/PDR reset, 2 = Independt watchdog reset, 3 = windows watchdog reset, 4 = low power reset, 5 = POR/PDR reset, 6 = Normal transmission, 7 = Button reset
	ProtocolVersion    uint     // Version 0 -> 65535
	BatteryVoltage     uint     // 0 -> 65535 mV
	Resistance         []uint64 // 0 -> 4294967295 Ohm
	SoilMoisture       []uint32 // 0 -> 65535 kPa. If value is too low or too high (sensor not placed outdoors, cable broke, etc), the sensor data is considered invalid and may not be sent./
	Temperature        float32  // °C
}

func decodeSensefarmPayload(b []byte, p *payloadSensefarm) error {

	readable_bytes := string(b)
	readable_str := readable_bytes + " decoded as"
	decode_state := "decode_header_byte"
	println("Decoded string " + string(b))

	//	known_protocol := false //Check that we are decoding a protocol we actually know how to handle

	for i := 0; i < len(b); i++ { //The multisensor message are read byte by byte and parsed for information on each individual sensor and it's values.
		readable_str += ""
		switch decode_state {
		case "decode_header_byte":
			switch (b[i] & 0xFF) >> 3 {
			case 0x01: //  Temperature
				byteValue := byteToValue(b, i, 4)
				fmt.Println("0x01 = ", byteValue)
				p.Temperature = float32(byteValue)
			case 0x06: // Battery
				byteValue := byteToValue(b, i, 2)
				fmt.Println("0x06 = ", byteValue)
				p.BatteryVoltage = uint(byteValue)
			case 0x13: //Resistance
				byteValue := byteToValue(b, i, 4)
				fmt.Println("0x13 = ", byteValue)
				p.Resistance = append(p.Resistance, byteValue)
			case 0x15: // Soil moisture
				byteValue := byteToValue(b, i, 2)
				fmt.Println("0x15 = ", byteValue)
				p.SoilMoisture = append(p.SoilMoisture, uint32(byteValue))
			case 0x016: // Transmission reason
				byteValue := byteToValue(b, i, 1)
				fmt.Println("0x16 = ", byteValue)
				p.TransmissionReason = uint(byteValue)
			case 0x17: // Protocol version
				byteValue := byteToValue(b, i, 2)
				fmt.Println("0x17 = ", byteValue)
				p.ProtocolVersion = uint(byteValue)
			default:
			}
		}
	}

	return nil
}

func byteToValue(b []byte, pos int, number_of_bytes int) uint64 {
	var value uint64

	for j := 0; j < number_of_bytes; j++ {
		value = value * 256
		pos++
		value += uint64((b[pos] & 0xFF))
	}
	return value
}
