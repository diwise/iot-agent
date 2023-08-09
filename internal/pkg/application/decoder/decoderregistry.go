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
		"cube02":           sensefarm.BasicDecoder, // deprecated, use sensefarm
		"elsys":            elsys.Decoder,
		"elsys_codec":      elsys.Decoder, // deprecated, use elsys
		"enviot":           enviot.Decoder,
		"milesight":        milesight.Decoder,
		"milesight_am100":  milesight.Decoder, // deprecated, use milesight
		"niab-fls":         niab.FillLevelSensorDecoder,
		"presence":         sensative.PresenceDecoder, // deprecated, use sensative
		"qalcosonic":       qalcosonic.W1Decoder,
		"sensative":        sensative.Decoder,
		"sensefarm":        sensefarm.BasicDecoder,
		"strips_lora_ms_h": sensative.Decoder, // deprecated, use sensative
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
