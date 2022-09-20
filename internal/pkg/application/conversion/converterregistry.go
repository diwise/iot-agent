package conversion

import (
	"context"

	lwm2m "github.com/diwise/iot-core/pkg/lwm2m"
)

type ConverterRegistry interface {
	DesignateConverters(ctx context.Context, types []string) []MessageConverterFunc
}

type converterRegistry struct {
	registeredConverters map[string]MessageConverterFunc
}

func NewConverterRegistry() ConverterRegistry {

	converters := map[string]MessageConverterFunc{
		lwm2m.Temperature:        Temperature,
		lwm2m.AirQuality:         AirQuality,
		lwm2m.Presence:           Presence,
		lwm2m.Watermeter:         Watermeter,
		"urn:oma:lwm2m:ext:3411": Battery,
		"urn:oma:lwm2m:ext:3304": Humidity,
	}

	return &converterRegistry{
		registeredConverters: converters,
	}
}

func (c *converterRegistry) DesignateConverters(ctx context.Context, types []string) []MessageConverterFunc {
	converters := []MessageConverterFunc{}

	for _, t := range types {
		mc, exist := c.registeredConverters[t]
		if exist {
			converters = append(converters, mc)
		}
	}

	return converters
}
