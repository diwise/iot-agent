package talkpool

import (
	"errors"
	"math"
)

var (
	ErrUnsupportedPort = errors.New("oy1210: unsupported fPort, expected 2")
	ErrInvalidLength   = errors.New("oy1210: payload must contain exactly 5 bytes")
)

type Oy1210Data struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	CO2         int     `json:"co2"`
}

func DecodeOy1210Payload(bytes []byte, port int) (*Oy1210Data, error) {
	if port != 2 {
		return nil, ErrUnsupportedPort
	}

	if len(bytes) != 5 {
		return nil, ErrInvalidLength
	}

	tempRaw := (int(bytes[0]) << 4) | (int(bytes[2]) >> 4)
	humidityRaw := (int(bytes[1]) << 4) | int(bytes[2])&0x0F
	co2Raw := (int(bytes[3]) << 8) | int(bytes[4])

	return &Oy1210Data{
		Temperature: roundToOneDecimal(float64(tempRaw)/10.0 - 80.0),
		Humidity:    roundToOneDecimal(float64(humidityRaw)/10.0 - 25.0),
		CO2:         co2Raw,
	}, nil
}

func roundToOneDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}
