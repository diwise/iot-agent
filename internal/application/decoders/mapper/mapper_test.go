package mapper

import (
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/matryer/is"
)

var mapperTime = time.Date(2024, 2, 28, 11, 21, 59, 0, time.UTC)

func mapperGroups() map[string]Group {
	return map[string]Group{
		"temp":     {Name: "temp", URN: "urn:oma:lwm2m:ext:3303"},
		"humidity": {Name: "humidity", URN: "urn:oma:lwm2m:ext:3304"},
		"device":   {Name: "device", URN: "urn:oma:lwm2m:ext:3"},
		"air":      {Name: "air", URN: "urn:oma:lwm2m:ext:3428"},
		"checkin":  {Name: "checkin", URN: "urn:oma:lwm2m:ext:3"},
		"cond":     {Name: "cond", URN: "urn:oma:lwm2m:ext:3327"},
	}
}

func TestMapTemperature(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"temperature": 24.1264}
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(len(res.Results), 1)
	is.Equal(res.Results[0].Status, RuleMatched)

	temp, ok := res.Objects[0].(lwm2m.Temperature)
	is.True(ok)
	is.Equal(temp.SensorValue, 24.126)

	pack := lwm2m.ToPack(res.Objects[0])
	is.Equal(pack[1].Name, "5700")
}

func TestMapDeviceMultiResource(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"vdd": 3636.0, "battery_level": 100.0}
	rules := []Rule{
		{ID: "v", From: "vdd", Group: "device", Resource: "7"},
		{ID: "b", From: "battery_level", Group: "device", Resource: "9"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)

	d, ok := res.Objects[0].(lwm2m.Device)
	is.True(ok)
	is.Equal(*d.PowerSourceVoltage, 3636)
	is.Equal(*d.BatteryLevel, 100)
}

func TestMapMissingSkipsSilently(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"temperature": 20.0}
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
		{ID: "h", From: "humidity", Group: "humidity", Resource: "5700"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(res.Results[0].Status, RuleMatched)
	is.Equal(res.Results[1].Status, RuleMissing)
}

func TestMapDisabledRule(t *testing.T) {
	is := is.New(t)

	disabled := false
	data := types.GenericPayload{"temperature": 20.0}
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700", Enabled: &disabled},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 0)
	is.Equal(res.Results[0].Status, RuleDisabled)
}

func TestMapTransformErrorDropsGroupOnly(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"temperature": "hot", "humidity": 40.0}
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
		{ID: "h", From: "humidity", Group: "humidity", Resource: "5700"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(res.Results[0].Status, RuleInvalid)
	is.Equal(res.Results[0].ErrorCode, ErrCodeType)
	is.Equal(res.Results[1].Status, RuleMatched)

	_, ok := res.Objects[0].(lwm2m.Humidity)
	is.True(ok)
}

func TestMapAliasFallback(t *testing.T) {
	is := is.New(t)

	// soundAvg present wins over soundPeak alias.
	data := types.GenericPayload{"soundAvg": 50.0, "soundPeak": 70.0}
	rules := []Rule{
		{ID: "s", From: "soundAvg", Aliases: []string{"soundPeak"}, Group: "temp", Resource: "5700"},
	}
	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(res.Results[0].Status, RuleMatched)
	is.Equal(res.Results[0].Path, "soundAvg")

	// soundAvg absent falls back to soundPeak.
	data = types.GenericPayload{"soundPeak": 70.0}
	res = MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(res.Results[0].Path, "soundPeak")
	temp := res.Objects[0].(lwm2m.Temperature)
	is.Equal(temp.SensorValue, 70.0)
}

func TestMapConst(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"battery": 87.0}
	rules := []Rule{
		{ID: "b", From: "battery", Group: "device", Resource: "9"},
		{ID: "t", Group: "device", Resource: "17", Const: &ConstValue{V: "acme"}},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(res.Results[1].Status, RuleMatched)
	is.Equal(res.Results[1].Path, "(const)")

	d, ok := res.Objects[0].(lwm2m.Device)
	is.True(ok)
	is.Equal(*d.BatteryLevel, 87)
	is.Equal(*d.DeviceType, "acme")
}

func TestMapAmbiguousIsInvalid(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"Temperature": 1.0, "TEMPERATURE": 2.0}
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 0)
	is.Equal(res.Results[0].Status, RuleInvalid)
	is.Equal(res.Results[0].ErrorCode, ErrCodeAmbiguous)
}

func TestMapListExpansion(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"resistances": []any{1000.0, 2000.0}}
	rules := []Rule{
		{ID: "r", List: "resistances", Group: "cond", Resource: "5700", Transforms: []TransformStep{{Name: "reciprocal"}}},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 2)
	is.Equal(len(res.Results), 2)
	is.Equal(res.Results[0].Index, 0)
	is.Equal(res.Results[1].Index, 1)

	first := res.Objects[0].(lwm2m.Conductivity)
	second := res.Objects[1].(lwm2m.Conductivity)
	is.Equal(first.SensorValue, 0.001)
	is.Equal(second.SensorValue, 0.0005)

	// A failing element drops only its own object.
	data = types.GenericPayload{"resistances": []any{1000.0, 0.0}}
	res = MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	is.Equal(res.Results[1].Status, RuleInvalid)
}

func TestMapHeaderOnly(t *testing.T) {
	is := is.New(t)

	rules := []Rule{
		{ID: "c", From: "check_in", Group: "checkin", HeaderOnly: true},
	}

	res := MapToLWM2M("dev", mapperTime, types.GenericPayload{"check_in": true}, mapperGroups(), rules)
	is.Equal(len(res.Objects), 1)
	_, ok := res.Objects[0].(lwm2m.Device)
	is.True(ok)

	// False flag means no header.
	res = MapToLWM2M("dev", mapperTime, types.GenericPayload{"check_in": false}, mapperGroups(), rules)
	is.Equal(len(res.Objects), 0)
	is.Equal(res.Results[0].Status, RuleMissing)
}

func TestMapCreatorGating(t *testing.T) {
	is := is.New(t)

	// A matched optional-only rule builds no object without a creator.
	data := types.GenericPayload{}
	rules := []Rule{
		{ID: "u", Group: "temp", Resource: "5701", Const: &ConstValue{V: "Cel"}},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 0)
	is.Equal(res.Results[0].Status, RuleMatched)
}

func TestMapUnknownGroupRobust(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"temperature": 20.0}
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "nope", Resource: "5700"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 0)
	is.Equal(res.Results[0].Status, RuleInvalid)
	is.Equal(res.Results[0].ErrorCode, ErrCodeGroup)
}

func TestMapDeterministicOrder(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"temperature": 20.0, "humidity": 40.0, "vdd": 3600.0}
	rules := []Rule{
		{ID: "h", From: "humidity", Group: "humidity", Resource: "5700"},
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
		{ID: "v", From: "vdd", Group: "device", Resource: "7"},
	}

	res := MapToLWM2M("dev", mapperTime, data, mapperGroups(), rules)
	is.Equal(len(res.Objects), 3)
	_, ok := res.Objects[0].(lwm2m.Humidity)
	is.True(ok)
	_, ok = res.Objects[1].(lwm2m.Temperature)
	is.True(ok)
	_, ok = res.Objects[2].(lwm2m.Device)
	is.True(ok)
}
