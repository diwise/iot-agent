package decoder

import (
	"context"
	"strings"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/niab"
)

type DecoderRegistry interface {
	GetDecoderForSensorType(ctx context.Context, sensorType string) MessageDecoderFunc
}

type decoderRegistry struct {
	registeredDecoders map[string]MessageDecoderFunc
}

func NewDecoderRegistry() DecoderRegistry {

	Decoders := map[string]MessageDecoderFunc{
		"qalcosonic":       QalcosonicW1,
		"presence":         PresenceDecoder,
		"elsys_codec":      ElsysDecoder,
		"enviot":           EnviotDecoder,
		"tem_lab_14ns":     SenlabTBasicDecoder,
		"strips_lora_ms_h": SensativeDecoder,
		"cube02":           SensefarmBasicDecoder,
		"milesight_am100":  MilesightDecoder,
		"niab-fls":         niab.FillLevelSensorDecoder,
	}

	return &decoderRegistry{
		registeredDecoders: Decoders,
	}
}

func (c *decoderRegistry) GetDecoderForSensorType(ctx context.Context, sensorType string) MessageDecoderFunc {

	if d, ok := c.registeredDecoders[strings.ToLower(sensorType)]; ok {
		return d
	}

	return DefaultDecoder
}
