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
		"qalcosonic":       Qalcosonic_Auto,
		"qalcosonic_w1h":   Qalcosonic_w1h,
		"qalcosonic_w1t":   Qalcosonic_w1t,
		"qalcosonic_w1e":   Qalcosonic_w1e,
		"presence":         PresenceDecoder,
		"elsys_codec":      ElsysDecoder,
		"enviot":           EnviotDecoder,
		"tem_lab_14ns":     SenlabTBasicDecoder,
		"strips_lora_ms_h": SensativeDecoder,
		"cube02":           SensefarmBasicDecoder,
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
