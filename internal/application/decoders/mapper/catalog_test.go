package mapper

import (
	"testing"
	"time"

	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

var testTime = time.Date(2024, 2, 28, 11, 21, 59, 0, time.UTC)

func TestKnownURN(t *testing.T) {
	is := is.New(t)

	is.True(KnownURN("urn:oma:lwm2m:ext:3303"))
	is.True(KnownURN("urn:oma:lwm2m:ext:3"))
	is.True(!KnownURN("urn:oma:lwm2m:ext:3411"))
	is.True(!KnownURN("nope"))
}

func TestAssembleTemperature(t *testing.T) {
	is := is.New(t)

	obj, err := AssembleObject("urn:oma:lwm2m:ext:3303", "dev", testTime, map[string]any{"5700": 24.126})
	is.NoErr(err)
	temp, ok := obj.(lwm2m.Temperature)
	is.True(ok)
	is.Equal(temp.SensorValue, 24.126)

	pack := lwm2m.ToPack(obj)
	is.Equal(len(pack), 2)
	is.Equal(pack[1].Name, "5700")
	is.Equal(*pack[1].Value, 24.126)
}

func TestAssembleDeviceMultiResource(t *testing.T) {
	is := is.New(t)

	obj, err := AssembleObject("urn:oma:lwm2m:ext:3", "dev", testTime, map[string]any{"7": 3636.0, "9": 100.0})
	is.NoErr(err)
	d, ok := obj.(lwm2m.Device)
	is.True(ok)
	is.Equal(*d.PowerSourceVoltage, 3636)
	is.Equal(*d.BatteryLevel, 100)
}

func TestAssembleAirQualityPartial(t *testing.T) {
	is := is.New(t)

	obj, err := AssembleObject("urn:oma:lwm2m:ext:3428", "dev", testTime, map[string]any{"17": 826.0})
	is.NoErr(err)
	aq, ok := obj.(lwm2m.AirQuality)
	is.True(ok)
	is.Equal(*aq.CO2, 826.0)
	is.True(aq.PM10 == nil)
}

func TestAssembleDigitalInputWithCounter(t *testing.T) {
	is := is.New(t)

	obj, err := AssembleObject("urn:oma:lwm2m:ext:3200", "dev", testTime, map[string]any{"5500": true, "5501": 42.0})
	is.NoErr(err)
	di, ok := obj.(lwm2m.DigitalInput)
	is.True(ok)
	is.True(di.DigitalInputState)
	is.Equal(*di.DigitalInputCounter, 42)
}

func TestAssembleErrors(t *testing.T) {
	is := is.New(t)

	_, err := AssembleObject("urn:oma:lwm2m:ext:9999", "dev", testTime, map[string]any{})
	is.True(err != nil)

	_, err = AssembleObject("urn:oma:lwm2m:ext:3303", "dev", testTime, map[string]any{"9999": 1.0})
	is.True(err != nil)

	_, err = AssembleObject("urn:oma:lwm2m:ext:3303", "dev", testTime, map[string]any{})
	is.True(err != nil)

	_, err = AssembleObject("urn:oma:lwm2m:ext:3303", "dev", testTime, map[string]any{"5700": "hot"})
	is.True(err != nil)

	_, err = AssembleObject("urn:oma:lwm2m:ext:3200", "dev", testTime, map[string]any{"5500": 1.0})
	is.True(err != nil)
}
