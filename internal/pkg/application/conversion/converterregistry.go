package conversion

import (
	"context"
)

const (
	AirQualityURN   string = "urn:oma:lwm2m:ext:3428"
	ConductivityURN string = "urn:oma:lwm2m:ext:3327"
	DigitalInputURN string = "urn:oma:lwm2m:ext:3200"
	EnergyURN       string = "urn:oma:lwm2m:ext:3331"
	HumidityURN     string = "urn:oma:lwm2m:ext:3304"
	IlluminanceURN  string = "urn:oma:lwm2m:ext:3301"
	PeopleCountURN  string = "urn:oma:lwm2m:ext:3434"
	PowerURN        string = "urn:oma:lwm2m:ext:3328"
	PresenceURN     string = "urn:oma:lwm2m:ext:3302"
	PressureURN     string = "urn:oma:lwm2m:ext:3323"
	TemperatureURN  string = "urn:oma:lwm2m:ext:3303"
	DistanceURN     string = "urn:oma:lwm2m:ext:3330"
	WatermeterURN   string = "urn:oma:lwm2m:ext:3424"
)

type ConverterRegistry interface {
	DesignateConverters(ctx context.Context, types []string) []MessageConverterFunc
}

type converterRegistry struct {
	registeredConverters map[string]MessageConverterFunc
}

func NewConverterRegistry() ConverterRegistry {

	converters := map[string]MessageConverterFunc{
		AirQualityURN:   AirQuality,
		ConductivityURN: Conductivity,
		DigitalInputURN: DigitalInput,
		DistanceURN:     Distance,
		HumidityURN:     Humidity,
		IlluminanceURN:  Illuminance,
		PeopleCountURN:  PeopleCount,
		PresenceURN:     Presence,
		PressureURN:     Pressure,
		TemperatureURN:  Temperature,
		WatermeterURN:   Watermeter,
	}

	return &converterRegistry{
		registeredConverters: converters,
	}
}

func (c *converterRegistry) DesignateConverters(ctx context.Context, types []string) []MessageConverterFunc {
	converters := []MessageConverterFunc{}

	for _, t := range types {
		if mc, exist := c.registeredConverters[t]; exist {
			converters = append(converters, mc)
		}
	}

	return converters
}
