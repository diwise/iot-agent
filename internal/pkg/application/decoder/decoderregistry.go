package decoder

import (
	"context"
	"strings"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/elsys"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/enviot"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/milesight"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/niab"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/qalcosonic"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/senlabt"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/sensative"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/sensefarm"
)

type DecoderRegistry interface {
	GetDecoderForSensorType(ctx context.Context, sensorType string) MessageDecoderFunc
}

type decoderRegistry struct {
	registeredDecoders map[string]MessageDecoderFunc
}

func NewDecoderRegistry() DecoderRegistry {

	Decoders := map[string]MessageDecoderFunc{
		"cube02":           sensefarm.BasicDecoder,
		"elsys_codec":      elsys.Decoder,
		"enviot":           enviot.Decoder,
		"milesight_am100":  milesight.Decoder,
		"niab-fls":         niab.FillLevelSensorDecoder,
		"presence":         sensative.PresenceDecoder,
		"qalcosonic":       qalcosonic.W1Decoder,
		"strips_lora_ms_h": sensative.Decoder,
		"tem_lab_14ns":     senlabt.BasicDecoder,
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
