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
		"urn:oma:lwm2m:ext:3323": Pressure,     //lwm2m.Pressure:      Pressure,
		"urn:oma:lwm2m:ext:3327": Conductivity, //lwm2m.Conductivity:	Conductivity
		//		"w3org.SoilHumidity": SoilMoisture, // "http://purl.org/iot/vocab/m3-lite#SoilHumidity", 	<todo> impl senare från core
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
