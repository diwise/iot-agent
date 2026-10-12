package mapper

import (
	"testing"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/matryer/is"
)

func TestApplyStatusBattery(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"battery_level": 87.0}
	res := ApplyStatus(data, StatusMapping{Battery: StatusSource{Path: "battery_level"}})
	is.Equal(len(res.Issues), 0)
	is.Equal(*res.Battery, 87)

	// Missing and null mean absent without issues.
	res = ApplyStatus(types.GenericPayload{}, StatusMapping{Battery: StatusSource{Path: "battery_level"}})
	is.True(res.Battery == nil)
	is.Equal(len(res.Issues), 0)

	res = ApplyStatus(types.GenericPayload{"battery_level": nil}, StatusMapping{Battery: StatusSource{Path: "battery_level"}})
	is.True(res.Battery == nil)
	is.Equal(len(res.Issues), 0)

	// Out-of-range values are rejected, never clamped.
	res = ApplyStatus(types.GenericPayload{"battery_level": 3636.0}, StatusMapping{Battery: StatusSource{Path: "battery_level"}})
	is.True(res.Battery == nil)
	is.Equal(len(res.Issues), 1)
	is.Equal(res.Issues[0].Code, "battery")

	// Empty path means absent.
	res = ApplyStatus(data, StatusMapping{})
	is.True(res.Battery == nil)
	is.Equal(len(res.Issues), 0)
}

func TestApplyStatusBatteryChain(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{"voltage_mv": 3500.0}
	res := ApplyStatus(data, StatusMapping{
		Battery: StatusSource{Path: "voltage_mv", Transforms: []TransformStep{{Name: "vdd_to_soc"}}},
	})
	is.Equal(len(res.Issues), 0)
	is.Equal(*res.Battery, 85)
}

func TestApplyStatusCodeAndMessages(t *testing.T) {
	is := is.New(t)

	data := types.GenericPayload{
		"code":     "E1",
		"messages": []any{"a", "b"},
	}
	res := ApplyStatus(data, StatusMapping{Code: StatusSource{Path: "code"}, Messages: StatusSource{Path: "messages"}})
	is.Equal(res.Code, "E1")
	is.Equal(res.Messages, []string{"a", "b"})
	is.Equal(len(res.Issues), 0)

	// Integral numbers are accepted as codes.
	res = ApplyStatus(types.GenericPayload{"code": 2.0}, StatusMapping{Code: StatusSource{Path: "code"}})
	is.Equal(res.Code, "2")

	// A single string is accepted as one message.
	res = ApplyStatus(types.GenericPayload{"messages": "solo"}, StatusMapping{Messages: StatusSource{Path: "messages"}})
	is.Equal(res.Messages, []string{"solo"})

	// Wrong types are reported.
	res = ApplyStatus(
		types.GenericPayload{"code": true, "messages": []any{"ok", 1.0}},
		StatusMapping{Code: StatusSource{Path: "code"}, Messages: StatusSource{Path: "messages"}},
	)
	is.Equal(res.Code, "")
	is.True(res.Messages == nil)
	is.Equal(len(res.Issues), 2)
}
