package decoders

import (
	"context"
	"strings"

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

type DecoderFunc func(ctx context.Context, deviceID string, e types.SensorEvent) ([]lwm2m.Lwm2mObject, error)

type Registry interface {
	GetDecoderForSensorType(ctx context.Context, sensorType string) DecoderFunc
}

type registryImpl struct {
	decoders map[string]DecoderFunc
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

	return &registryImpl{
		decoders: decoders,
	}
}

func (c *registryImpl) GetDecoderForSensorType(ctx context.Context, sensorType string) DecoderFunc {
	if d, ok := c.decoders[strings.ToLower(sensorType)]; ok {
		return d
	}

	return defaultdecoder.DefaultDecoder
}
