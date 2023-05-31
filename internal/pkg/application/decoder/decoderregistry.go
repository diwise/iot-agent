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
		"cube02":           sensefarm.SensefarmBasicDecoder,
		"elsys_codec":      elsys.ElsysDecoder,
		"enviot":           enviot.EnviotDecoder,
		"milesight_am100":  milesight.MilesightDecoder,
		"niab-fls":         niab.FillLevelSensorDecoder,
		"presence":         sensative.PresenceDecoder,
		"qalcosonic":       qalcosonic.QalcosonicW1,
		"strips_lora_ms_h": sensative.SensativeDecoder,
		"tem_lab_14ns":     senlabt.SenlabTBasicDecoder,
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
