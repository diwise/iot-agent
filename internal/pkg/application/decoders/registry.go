package decoders

import (
	"context"
	"fmt"
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
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type DecoderFunc func(ctx context.Context, e types.Event) (types.SensorPayload, error)
type ConverterFunc func(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error)

type Registry interface {
	Get(ctx context.Context, sensorType string) (DecoderFunc, ConverterFunc, bool)
}

type registryImpl struct {
	decoders   map[string]DecoderFunc
	converters map[string]ConverterFunc
}

func NewRegistry() Registry {
	decoders := map[string]DecoderFunc{
		"airquality": airquality.Decoder,
		"axsensor":   axsensor.Decoder,
		"enviot":     enviot.Decoder,
		"niab-fls":   niab.Decoder,

		"qalcosonic":      qalcosonic.Decoder,
		"qalcosonic/w1t":  qalcosonic.DecoderW1t,
		"qalcosonic/w1h":  qalcosonic.DecoderW1h,
		"qalcosonic/w1e":  qalcosonic.DecoderW1e,
		
		"vegapuls_air_41": vegapuls.Decoder,

		"elt_2_hp":        elsys.Decoder,
		"elsys":           elsys.Decoder,
		"elsys/elt/sht3x": elsys.Decoder,
		"elsys_codec":     elsys.Decoder, // deprecated, use elsys

		"milesight":       milesight.Decoder,
		"milesight_am100": milesight.Decoder, // deprecated, use milesight

		"senlabt":      senlabt.Decoder,
		"tem_lab_14ns": senlabt.Decoder, // deprecated, use senlabt

		"sensefarm": sensefarm.Decoder,
		"cube02":    sensefarm.Decoder, // deprecated, use sensefarm

		"sensative":        sensative.Decoder,
		"strips_lora_ms_h": sensative.Decoder, // deprecated, use sensative
		"presence":         sensative.Decoder, // deprecated, use sensative
	}

	converters := map[string]ConverterFunc{
		"airquality": airquality.Converter,
		"axsensor":   axsensor.Converter,
		"enviot":     enviot.Converter,
		"niab-fls":   niab.Converter,

		"qalcosonic":     qalcosonic.Converter,
		"qalcosonic/w1t": qalcosonic.Converter,
		"qalcosonic/w1h": qalcosonic.Converter,
		"qalcosonic/w1e": qalcosonic.Converter,

		"vegapuls_air_41": vegapuls.Converter,

		"elt_2_hp":        elsys.Converter,
		"elsys":           elsys.Converter,
		"elsys/elt/sht3x": elsys.ConverterEltSht3x,
		"elsys_codec":     elsys.Converter, // deprecated, use elsys

		"milesight":       milesight.Converter,
		"milesight_am100": milesight.Converter, // deprecated, use milesight

		"senlabt":      senlabt.Converter,
		"tem_lab_14ns": senlabt.Converter, // deprecated, use senlabt

		"sensefarm": sensefarm.Converter,
		"cube02":    sensefarm.Converter, // deprecated, use sensefarm

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

	log := logging.GetFromContext(ctx)

	errCounter, err := otel.Meter("iot-agent/decoding").Int64Counter(
		"diwise.decoding.errors.total",
		metric.WithUnit("1"),
		metric.WithDescription("Total number of errors"),
	)

	if err != nil {
		log.Error("failed to create otel message counter", "err", err.Error())
	}

	decoder := func(fn DecoderFunc) DecoderFunc {
		messageCounter, err := otel.Meter("iot-agent/decoding").Int64Counter(
			"diwise.decoding."+sensorType+".total",
			metric.WithUnit("1"),
			metric.WithDescription(fmt.Sprintf("Total number of decoded %s payloads", sensorType)),
		)

		if err != nil {
			log.Error("failed to create otel message counter", "err", err.Error())
		}

		return func(ctx context.Context, e types.Event) (types.SensorPayload, error) {
			p, err := fn(ctx, e)
			if err != nil {
				errCounter.Add(ctx, 1)
				return nil, err
			}

			messageCounter.Add(ctx, 1)
			return p, nil
		}
	}

	converter := func(fn ConverterFunc) ConverterFunc {
		return func(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
			return fn(ctx, deviceID, payload, ts)
		}
	}

	if d, ok := c.decoders[strings.ToLower(sensorType)]; ok {
		if c, ok := c.converters[strings.ToLower(sensorType)]; ok {
			return decoder(d), converter(c), true
		} else {
			return decoder(d), converter(defaultdecoder.Converter), false
		}
	}

	return defaultdecoder.Decoder, defaultdecoder.Converter, false
}
