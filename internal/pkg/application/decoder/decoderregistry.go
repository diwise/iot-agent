package decoder

import (
	"context"
	"strings"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/airquality"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/axsensor"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/elsys"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/enviot"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/milesight"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/niab"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/qalcosonic"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/senlabt"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/sensative"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/sensefarm"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/vegapuls"
)

type DecoderRegistry interface {
	GetDecoderForSensorType(ctx context.Context, sensorType string) MessageDecoderFunc
}

type decoderRegistry struct {
	registeredDecoders map[string]MessageDecoderFunc
}

func NewDecoderRegistry() DecoderRegistry {
	Decoders := map[string]MessageDecoderFunc{
		"airquality":       airquality.Decoder,
		"axsensor":         axsensor.Decoder,
		"cube02":           sensefarm.Decoder, // deprecated, use sensefarm
		"elsys":            elsys.Decoder,
		"elsys_codec":      elsys.Decoder, // deprecated, use elsys
		"elt_2_hp":         elsys.Decoder,
		"enviot":           enviot.Decoder,
		"milesight":        milesight.Decoder,
		"milesight_am100":  milesight.Decoder, // deprecated, use milesight
		"niab-fls":         niab.Decoder,
		"presence":         sensative.Decoder, // deprecated, use sensative
		"qalcosonic":       qalcosonic.Decoder,
		"senlabt":          senlabt.Decoder,
		"sensative":        sensative.Decoder,
		"sensefarm":        sensefarm.Decoder,
		"strips_lora_ms_h": sensative.Decoder, // deprecated, use sensative
		"tem_lab_14ns":     senlabt.Decoder,   // deprecated, use senlabt
		"vegapuls_air_41":  vegapuls.Decoder,
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
