package x2

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type X2ClimateTelemetry struct {
	// Header-ish
	MsgType uint8 `json:"msg_type"`
	Flags   uint8 `json:"flags"`
	Seq     uint8 `json:"seq"`
	BattRaw uint8 `json:"batt_raw"`

	// Measurements (best-effort)
	TempC      float64 `json:"temp_c"`
	HumidityRH float64 `json:"humidity_rh"`
	VoltageV   float64 `json:"voltage_v"`

	// Status/counters
	Status  uint8  `json:"status"`
	Counter uint32 `json:"counter"`

	// Raw fields for debugging/verification
	Raw struct {
		TempI16     int16  `json:"temp_i16"`
		HumU16      uint16 `json:"hum_u16"`
		VoltageU16  uint16 `json:"voltage_u16"`
		ReservedU16 uint16 `json:"reserved_u16"`
	} `json:"raw"`
}

func DecodeX2Climate(b []byte) (X2ClimateTelemetry, error) {
	var out X2ClimateTelemetry
	if len(b) != 19 {
		return out, fmt.Errorf("unexpected length: got %d bytes, want 19", len(b))
	}

	// Bytes (index):  0   1   2   3   4  5   6  7   8  9  10 11  12 13  14  15 16 17 18
	// Example:       03  11  01  14  8C FB 00 00 0C A4 00 C5 08 78 28 00 00 00 01

	out.MsgType = b[0]
	out.Flags = b[1]
	out.Seq = b[2]
	out.BattRaw = b[3]

	// Temp: bytes 4-5 look little-endian signed (8C FB -> 0xFB8C = -1140 => -11.40°C if /100)
	tempI16 := int16(binary.LittleEndian.Uint16(b[4:6]))
	out.Raw.TempI16 = tempI16
	out.TempC = float64(tempI16) / 100.0

	// Reserved/unknown: bytes 6-7 (often 0)
	out.Raw.ReservedU16 = binary.BigEndian.Uint16(b[6:8])

	// Humidity: bytes 8-9.
	// For your sample, big-endian /100 => 0x0CA4 = 3236 => 32.36% (plausible).
	// If future samples look off, try little-endian /1000 as an alternative.
	humU16 := binary.BigEndian.Uint16(b[8:10])
	out.Raw.HumU16 = humU16
	out.HumidityRH = float64(humU16) / 100.0

	// Bytes 10-11: unknown (kept raw only for now)
	_ = binary.BigEndian.Uint16(b[10:12])

	// Voltage: bytes 12-13.
	// Sample big-endian 0x0878 = 2168 mV => 2.168V (low-but-possible).
	// If that seems wrong, try little-endian with another scale.
	voltU16 := binary.BigEndian.Uint16(b[12:14])
	out.Raw.VoltageU16 = voltU16
	out.VoltageV = float64(voltU16) / 1000.0

	// Status: byte 14
	out.Status = b[14]

	// Counter: bytes 15-18 big-endian uint32 (00 00 00 01 => 1)
	out.Counter = binary.BigEndian.Uint32(b[15:19])

	// Basic sanity checks (optional)
	if out.HumidityRH < 0 || out.HumidityRH > 110 {
		return out, errors.New("decoded humidity out of plausible range; check endianness/scale")
	}
	if out.VoltageV < 0.5 || out.VoltageV > 5.5 {
		return out, errors.New("decoded voltage out of plausible range; check endianness/scale")
	}

	return out, nil
}
