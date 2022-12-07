package decoder

import (
	"context"
	"strings"
)

type DecoderRegistry interface {
	GetDecoderForSensorType(ctx context.Context, sensorType string) MessageDecoderFunc
}

type decoderRegistry struct {
	registeredDecoders map[string]MessageDecoderFunc
}

func NewDecoderRegistry() DecoderRegistry {

	Decoders := map[string]MessageDecoderFunc{
		"qalcosonic":       QalcosonicAuto,
		"qalcosonic_w1h":   QalcosonicW1h,
		"qalcosonic_w1t":   QalcosonicW1t,
		"qalcosonic_w1e":   QalcosonicW1e,
		"presence":         PresenceDecoder,
		"elsys_codec":      ElsysDecoder,
		"enviot":           EnviotDecoder,
		"tem_lab_14ns":     SenlabTBasicDecoder,
		"strips_lora_ms_h": SensativeDecoder,
		"cube02":           SensefarmBasicDecoder,
		"milesight am100":  MilesightDecoder,
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
