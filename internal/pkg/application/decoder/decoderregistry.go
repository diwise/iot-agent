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
		"tem_lab_14ns":     SenlabTBasicDecoder,
		"elsys_codec":      ElsysDecoder,
		"strips_lora_ms_h": SensativeDecoder,
		"enviot":           EnviotDecoder,
		"presence":         PresenceDecoder,
		"qalcosonic":       AxiomaWatermeteringDecoder,
		"qalcosonic_w1h":   Qalcosonic_w1h,
		"qalcosonic_w1t":   Qalcosonic_w1t,
		"qalcosonic_w1e":   Qalcosonic_w1e,
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
