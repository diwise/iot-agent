package decoders

import (
	"context"
	"strings"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoders/airquality"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/axsensor"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/defaultdecoder"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/elsys"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/enviot"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/milesight"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/niab"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/qalcosonic"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/senlabt"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/sensative"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/sensefarm"
	"github.com/diwise/iot-agent/internal/pkg/application/decoders/vegapuls"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type DecoderFunc func(ctx context.Context, e types.Event) (any, error)
type ConverterFunc func(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error)

type Registry interface {
	Get(ctx context.Context, sensorType string) (DecoderFunc, ConverterFunc, bool)
	GetDecoder(ctx context.Context, sensorType string) DecoderFunc
	GetConverter(ctx context.Context, sensorType string) ConverterFunc
}

type registryImpl struct {
	decoders   map[string]DecoderFunc
	converters map[string]ConverterFunc
}

func NewRegistry() Registry {
	decoders := map[string]DecoderFunc{
		"airquality":      airquality.Decoder,
		"axsensor":        axsensor.Decoder,
		"elt_2_hp":        elsys.Decoder,
		"enviot":          enviot.Decoder,
		"niab-fls":        niab.Decoder,
		"qalcosonic":      qalcosonic.Decoder,
		"vegapuls_air_41": vegapuls.Decoder,

		"elsys":       elsys.Decoder,
		"elsys_codec": elsys.Decoder, // deprecated, use elsys

		"milesight":       milesight.Decoder,
		"milesight_am100": milesight.Decoder, // deprecated, use milesight

		"senlabt":      senlabt.Decoder,
		"tem_lab_14ns": senlabt.Decoder, // deprecated, use senlabt

		"cube02":    sensefarm.Decoder, // deprecated, use sensefarm
		"sensefarm": sensefarm.Decoder,

		"sensative":        sensative.Decoder,
		"strips_lora_ms_h": sensative.Decoder, // deprecated, use sensative
		"presence":         sensative.Decoder, // deprecated, use sensative
	}

	converters := map[string]ConverterFunc{
		"airquality":      airquality.Converter,
		"axsensor":        axsensor.Converter,
		"elt_2_hp":        elsys.Converter,
		"enviot":          enviot.Converter,
		"niab-fls":        niab.Converter,
		"qalcosonic":      qalcosonic.Converter,
		"vegapuls_air_41": vegapuls.Converter,

		"elsys":       elsys.Converter,
		"elsys_codec": elsys.Converter, // deprecated, use elsys

		"milesight":       milesight.Converter,
		"milesight_am100": milesight.Converter, // deprecated, use milesight

		"senlabt":      senlabt.Converter,
		"tem_lab_14ns": senlabt.Converter, // deprecated, use senlabt

		"cube02":    sensefarm.Converter, // deprecated, use sensefarm
		"sensefarm": sensefarm.Converter,

		"sensative":        sensative.Converter,
		"strips_lora_ms_h": sensative.Converter, // deprecated, use sensative
		"presence":         sensative.Converter, // deprecated, use sensative
	}

	return &registryImpl{
		decoders:   decoders,
		converters: converters,
	}
}

func (c *registryImpl) Get(ctx context.Context, sensorType string) (DecoderFunc, ConverterFunc, bool) {
	if d, ok := c.decoders[strings.ToLower(sensorType)]; ok {
		if c, ok := c.converters[strings.ToLower(sensorType)]; ok {
			return d, c, true
		} else {
			return d, defaultdecoder.Converter, false
		}
	}

	return defaultdecoder.Decoder, defaultdecoder.Converter, false
}

func (c *registryImpl) GetDecoder(ctx context.Context, sensorType string) DecoderFunc {
	if d, ok := c.decoders[strings.ToLower(sensorType)]; ok {
		return d
	}

	return defaultdecoder.Decoder
}

func (c *registryImpl) GetConverter(ctx context.Context, sensorType string) ConverterFunc {
	if d, ok := c.converters[strings.ToLower(sensorType)]; ok {
		return d
	}

	return defaultdecoder.Converter
}
