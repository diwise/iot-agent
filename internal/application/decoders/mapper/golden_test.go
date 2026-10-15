package mapper_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/application/decoders/elsys"
	"github.com/diwise/iot-agent/internal/application/decoders/mapper"
	"github.com/diwise/iot-agent/internal/application/decoders/milesight"
	"github.com/diwise/iot-agent/internal/application/decoders/talkpool"
	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/matryer/is"
)

// External test package on purpose: golden tests import sensor
// packages, and sensor adapters will import mapper after migration.
// An internal test package would create an import cycle then.

var goldenTime = time.Date(2024, 2, 28, 11, 21, 59, 0, time.UTC)

func goldenCtx() context.Context {
	return logging.NewContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func marshalMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// assertPacksEqual compares full SenML packs as JSON text: object
// count and order, base fields, names, units, value types, values
// and nil versus zero.
func assertPacksEqual(t *testing.T, legacyObjs, mappedObjs []lwm2m.Lwm2mObject) {
	t.Helper()
	is := is.New(t)
	is.Equal(len(mappedObjs), len(legacyObjs))

	legacyPacks := make([]string, 0, len(legacyObjs))
	for _, o := range legacyObjs {
		b, err := json.Marshal(lwm2m.ToPack(o))
		is.NoErr(err)
		legacyPacks = append(legacyPacks, string(b))
	}
	mappedPacks := make([]string, 0, len(mappedObjs))
	for _, o := range mappedObjs {
		b, err := json.Marshal(lwm2m.ToPack(o))
		is.NoErr(err)
		mappedPacks = append(mappedPacks, string(b))
	}
	is.Equal(mappedPacks, legacyPacks)
}

//go:fix inline
func f32p(v float32) *float32 { return new(v) }

//go:fix inline
func i8p(v int8) *int8 { return new(v) }

//go:fix inline
func u16p(v uint16) *uint16 { return new(v) }

//go:fix inline
func u8p(v uint8) *uint8 { return new(v) }

//go:fix inline
func u32p(v uint32) *uint32 { return new(v) }

//go:fix inline
func boolp(v bool) *bool { return new(v) }

//go:fix inline
func intp(v int) *int { return new(v) }

//go:fix inline
func floatp(v float64) *float64 { return new(v) }

//go:fix inline
func stringp(v string) *string { return new(v) }

// elsysGenericMap mirrors the future Elsys adapter: marshal the typed
// payload, then restore mapped float32 fields with direct conversion
// so the mapper sees exactly the legacy converter input.
func elsysGenericMap(p elsys.ElsysPayload) map[string]any {
	b, _ := json.Marshal(p)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	if p.Temperature != nil {
		m["temperature"] = float64(*p.Temperature)
	}
	if p.ExternalTemperature != nil {
		m["externalTemperature"] = float64(*p.ExternalTemperature)
	}
	if p.ExternalTemperature2 != nil {
		m["externalTemperature2"] = float64(*p.ExternalTemperature2)
	}
	return m
}

func elsysGroups() map[string]mapper.Group {
	urn := func(id string) string { return "urn:oma:lwm2m:ext:" + id }
	return map[string]mapper.Group{
		"temp-in":   {Name: "temp-in", URN: urn("3303"), ChannelStrategy: "elsys", Role: "internal"},
		"temp-ext1": {Name: "temp-ext1", URN: urn("3303"), ChannelStrategy: "elsys", Role: "extern1"},
		"temp-ext2": {Name: "temp-ext2", URN: urn("3303"), ChannelStrategy: "elsys", Role: "extern2"},
		"humidity":  {Name: "humidity", URN: urn("3304")},
		"light":     {Name: "light", URN: urn("3301")},
		"air":       {Name: "air", URN: urn("3428")},
		"device":    {Name: "device", URN: urn("3")},
		"presence":  {Name: "presence", URN: urn("3302")},
		"loudness":  {Name: "loudness", URN: urn("3324")},
		"digital":   {Name: "digital", URN: urn("3200")},
	}
}

func elsysRules() []mapper.Rule {
	return []mapper.Rule{
		{ID: "t", From: "temperature", Group: "temp-in", Resource: "5700"},
		{ID: "e1", From: "externalTemperature", Group: "temp-ext1", Resource: "5700"},
		{ID: "e2", From: "externalTemperature2", Group: "temp-ext2", Resource: "5700"},
		{ID: "h", From: "humidity", Group: "humidity", Resource: "5700"},
		{ID: "l", From: "light", Group: "light", Resource: "5700"},
		{ID: "c", From: "co2", Group: "air", Resource: "17"},
		{ID: "v", From: "vdd", Group: "device", Resource: "7"},
		{ID: "b", From: "vdd", Group: "device", Resource: "9", Transforms: []mapper.TransformStep{{Name: "vdd_to_soc"}}},
		{ID: "o", From: "occupancy", Group: "presence", Resource: "5500", Transforms: []mapper.TransformStep{{Name: "equals", Params: map[string]any{"value": 2}}}},
		{ID: "sa", From: "soundAvg", Aliases: []string{"soundPeak"}, Group: "loudness", Resource: "5700"},
		{ID: "sp", From: "soundPeak", Group: "loudness", Resource: "5602"},
		{ID: "d", From: "digital", Group: "digital", Resource: "5500"},
		{ID: "pa", From: "pulseAbs", Group: "digital", Resource: "5501"},
	}
}

func elsysSHT3XGroups() map[string]mapper.Group {
	g := elsysGroups()
	g["temp-in"] = mapper.Group{Name: "temp-in", URN: "urn:oma:lwm2m:ext:3303", ChannelStrategy: "elsys-sht3x", Role: "internal"}
	g["temp-ext1"] = mapper.Group{Name: "temp-ext1", URN: "urn:oma:lwm2m:ext:3303", ChannelStrategy: "elsys-sht3x", Role: "extern1"}
	g["temp-ext2"] = mapper.Group{Name: "temp-ext2", URN: "urn:oma:lwm2m:ext:3303", ChannelStrategy: "elsys-sht3x", Role: "extern2"}
	g["humidity"] = mapper.Group{Name: "humidity", URN: "urn:oma:lwm2m:ext:3304", ChannelStrategy: "elsys-sht3x", Role: "humidity"}
	g["pulse-humidity"] = mapper.Group{Name: "pulse-humidity", URN: "urn:oma:lwm2m:ext:3304", ChannelStrategy: "elsys-sht3x", Role: "pulse"}
	return g
}

func elsysSHT3XRules() []mapper.Rule {
	rules := elsysRules()
	return append(rules, mapper.Rule{
		ID: "ph", From: "pulse1", Group: "pulse-humidity", Resource: "5700",
		Transforms: []mapper.TransformStep{{Name: "scale", Params: map[string]any{"divisor": 10}}},
	})
}

func TestGoldenElsysERS(t *testing.T) {
	is := is.New(t)
	ctx := goldenCtx()

	raw, err := base64.StdEncoding.DecodeString("AQDuAhYEALIFAgYBxAcONA==")
	is.NoErr(err)
	event := types.Event{Payload: &types.Payload{Data: raw, FPort: 5}, Timestamp: goldenTime}

	legacyPayload, err := elsys.Decoder(ctx, event)
	is.NoErr(err)
	legacyObjects, err := elsys.Converter(ctx, "ers01", legacyPayload, goldenTime)
	is.NoErr(err)
	is.Equal(len(legacyObjects), 5)

	mapped := mapper.MapToLWM2M("ers01", goldenTime, elsysGenericMap(legacyPayload.(elsys.ElsysPayload)), elsysGroups(), elsysRules())
	matched := map[string]bool{}
	for _, r := range mapped.Results {
		if r.Status == mapper.RuleMatched {
			matched[r.RuleID] = true
		} else {
			is.Equal(r.Status, mapper.RuleMissing)
		}
	}
	for _, id := range []string{"t", "h", "l", "c", "v", "b"} {
		is.True(matched[id])
	}
	assertPacksEqual(t, legacyObjects, mapped.Objects)
}

func TestGoldenElsysTemperatureChannels(t *testing.T) {
	is := is.New(t)
	ctx := goldenCtx()

	cases := []struct {
		name    string
		payload elsys.ElsysPayload
		suffix  []string
	}{
		{"internal only", elsys.ElsysPayload{Temperature: f32p(21.5)}, []string{""}},
		{"extern1 only", elsys.ElsysPayload{ExternalTemperature: f32p(19.3)}, []string{""}},
		{"extern2 only", elsys.ElsysPayload{ExternalTemperature2: f32p(20.1)}, []string{""}},
		{"two externals share no suffix", elsys.ElsysPayload{ExternalTemperature: f32p(19.3), ExternalTemperature2: f32p(20.1)}, []string{"", ""}},
		{"internal and extern1", elsys.ElsysPayload{Temperature: f32p(21.5), ExternalTemperature: f32p(19.3)}, []string{"/0", "/1"}},
		{"internal and extern2", elsys.ElsysPayload{Temperature: f32p(21.5), ExternalTemperature2: f32p(20.1)}, []string{"/0", "/2"}},
		{"all three", elsys.ElsysPayload{Temperature: f32p(21.5), ExternalTemperature: f32p(19.3), ExternalTemperature2: f32p(20.1)}, []string{"/0", "/1", "/2"}},
	}

	for _, tc := range cases {
		legacyObjects, err := elsys.Converter(ctx, "dev", tc.payload, goldenTime)
		is.NoErr(err)
		mapped := mapper.MapToLWM2M("dev", goldenTime, elsysGenericMap(tc.payload), elsysGroups(), elsysRules())
		assertPacksEqual(t, legacyObjects, mapped.Objects)

		is.Equal(len(mapped.Objects), len(tc.suffix))
		for i, want := range tc.suffix {
			pack := lwm2m.ToPack(mapped.Objects[i])
			is.Equal(pack[0].BaseName, "dev"+want+"/3303/")
		}
	}
}

func TestGoldenElsysSHT3X(t *testing.T) {
	is := is.New(t)
	ctx := goldenCtx()

	payload := elsys.ElsysPayload{Humidity: i8p(50), Pulse: u16p(215)}
	legacyObjects, err := elsys.ConverterEltSht3x(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	is.Equal(len(legacyObjects), 2)

	mapped := mapper.MapToLWM2M("dev", goldenTime, elsysGenericMap(payload), elsysSHT3XGroups(), elsysSHT3XRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)

	is.Equal(lwm2m.ToPack(mapped.Objects[0])[0].BaseName, "dev/0/3304/")
	is.Equal(lwm2m.ToPack(mapped.Objects[1])[0].BaseName, "dev/1/3304/")

	// Humidity without pulse keeps no suffix.
	payload = elsys.ElsysPayload{Humidity: i8p(50)}
	legacyObjects, err = elsys.ConverterEltSht3x(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	mapped = mapper.MapToLWM2M("dev", goldenTime, elsysGenericMap(payload), elsysSHT3XGroups(), elsysSHT3XRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)
	is.Equal(len(mapped.Objects), 1)
}

func TestGoldenElsysObjects(t *testing.T) {
	is := is.New(t)
	ctx := goldenCtx()

	payload := elsys.ElsysPayload{
		Temperature:  f32p(23.5),
		VDD:          u16p(3625),
		DigitalInput: new(true),
		PulseAbs:     u32p(123),
		Occupancy:    u8p(2),
		SoundAvg:     u8p(50),
		SoundPeak:    u8p(70),
	}
	legacyObjects, err := elsys.Converter(ctx, "dev", payload, goldenTime)
	is.NoErr(err)

	mapped := mapper.MapToLWM2M("dev", goldenTime, elsysGenericMap(payload), elsysGroups(), elsysRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)

	// Peak alone fills both value and max.
	payload = elsys.ElsysPayload{SoundPeak: u8p(70)}
	legacyObjects, err = elsys.Converter(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	mapped = mapper.MapToLWM2M("dev", goldenTime, elsysGenericMap(payload), elsysGroups(), elsysRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)

	// PulseAbs without digital builds no object.
	payload = elsys.ElsysPayload{PulseAbs: u32p(123)}
	legacyObjects, err = elsys.Converter(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	mapped = mapper.MapToLWM2M("dev", goldenTime, elsysGenericMap(payload), elsysGroups(), elsysRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)
	is.Equal(len(mapped.Objects), 0)
}

func milesightGroups() map[string]mapper.Group {
	urn := func(id string) string { return "urn:oma:lwm2m:ext:" + id }
	return map[string]mapper.Group{
		"device":   {Name: "device", URN: urn("3")},
		"air":      {Name: "air", URN: urn("3428")},
		"distance": {Name: "distance", URN: urn("3330")},
		"humidity": {Name: "humidity", URN: urn("3304")},
		"temp":     {Name: "temp", URN: urn("3303")},
		"digital":  {Name: "digital", URN: urn("3200")},
	}
}

func milesightRules() []mapper.Rule {
	return []mapper.Rule{
		{ID: "b", From: "battery", Group: "device", Resource: "9"},
		{ID: "c", From: "co2", Group: "air", Resource: "17"},
		{ID: "d", From: "distance", Group: "distance", Resource: "5700"},
		{ID: "h", From: "humidity", Group: "humidity", Resource: "5700"},
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
		{ID: "m", From: "magnet_status", Group: "digital", Resource: "5500", Transforms: []mapper.TransformStep{{Name: "magnet_open_to_bool"}}},
	}
}

func TestGoldenMilesight(t *testing.T) {
	is := is.New(t)
	ctx := goldenCtx()

	payload := milesight.MilesightPayload{
		Battery:      new(95),
		CO2:          new(800),
		Distance:     new(12.5),
		Humidity:     new(45.5),
		Temperature:  new(22.3),
		Position:     new("normal"),
		MagnetStatus: new("open"),
	}
	legacyObjects, err := milesight.Converter(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	is.Equal(len(legacyObjects), 6)

	mapped := mapper.MapToLWM2M("dev", goldenTime, marshalMap(t, payload), milesightGroups(), milesightRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)

	// Partial report with only temperature.
	payload = milesight.MilesightPayload{Temperature: new(22.3)}
	legacyObjects, err = milesight.Converter(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	mapped = mapper.MapToLWM2M("dev", goldenTime, marshalMap(t, payload), milesightGroups(), milesightRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)
	is.Equal(len(mapped.Objects), 1)

	// Closed magnet maps to false.
	payload = milesight.MilesightPayload{MagnetStatus: new("close")}
	legacyObjects, err = milesight.Converter(ctx, "dev", payload, goldenTime)
	is.NoErr(err)
	mapped = mapper.MapToLWM2M("dev", goldenTime, marshalMap(t, payload), milesightGroups(), milesightRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)
}

func talkpoolGroups() map[string]mapper.Group {
	urn := func(id string) string { return "urn:oma:lwm2m:ext:" + id }
	return map[string]mapper.Group{
		"temp":     {Name: "temp", URN: urn("3303")},
		"humidity": {Name: "humidity", URN: urn("3304")},
		"air":      {Name: "air", URN: urn("3428")},
	}
}

func talkpoolRules() []mapper.Rule {
	return []mapper.Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
		{ID: "h", From: "humidity", Group: "humidity", Resource: "5700"},
		{ID: "c", From: "co2", Group: "air", Resource: "17"},
	}
}

func TestGoldenTalkpoolGo(t *testing.T) {
	is := is.New(t)
	ctx := goldenCtx()

	payload := talkpool.Oy1210Data{Temperature: 24.1, Humidity: 43.8, CO2: 826}
	legacyObjects, err := talkpool.ConverterOy1210(ctx, "dev", &payload, goldenTime)
	is.NoErr(err)
	is.Equal(len(legacyObjects), 3)

	mapped := mapper.MapToLWM2M("dev", goldenTime, marshalMap(t, payload), talkpoolGroups(), talkpoolRules())
	assertPacksEqual(t, legacyObjects, mapped.Objects)

	// The same rules match capitalized JS keys through case-insensitive
	// lookup, so one rule set serves both Go and JS decoders.
	jsShaped := map[string]any{"Temperature": 24.1, "Humidity": 43.8, "CO2": 826}
	mappedJS := mapper.MapToLWM2M("dev", goldenTime, jsShaped, talkpoolGroups(), talkpoolRules())
	assertPacksEqual(t, legacyObjects, mappedJS.Objects)
}
