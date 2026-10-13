package mapper

import (
	"testing"

	"github.com/matryer/is"
)

func TestKnownChannelStrategies(t *testing.T) {
	is := is.New(t)

	is.True(KnownChannelStrategy("elsys"))
	is.True(KnownChannelStrategy("elsys-sht3x"))
	is.True(!KnownChannelStrategy("nope"))

	roles, ok := StrategyRoles("elsys")
	is.True(ok)
	is.Equal(roles, []string{"internal", "extern1", "extern2"})

	_, ok = StrategyRoles("nope")
	is.True(!ok)

	is.True(ResolveChannels("nope", map[string]bool{}) == nil)
}

func TestElsysTemperatureChannels(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		name    string
		matched map[string]bool
		want    map[string]string
	}{
		{"internal only", map[string]bool{"internal": true}, map[string]string{"internal": ""}},
		{"extern1 only", map[string]bool{"extern1": true}, map[string]string{"extern1": ""}},
		{"extern2 only", map[string]bool{"extern2": true}, map[string]string{"extern2": ""}},
		{"two externals share no suffix", map[string]bool{"extern1": true, "extern2": true}, map[string]string{"extern1": "", "extern2": ""}},
		{"internal and extern1", map[string]bool{"internal": true, "extern1": true}, map[string]string{"internal": "/0", "extern1": "/1"}},
		{"internal and extern2", map[string]bool{"internal": true, "extern2": true}, map[string]string{"internal": "/0", "extern2": "/2"}},
		{"all three", map[string]bool{"internal": true, "extern1": true, "extern2": true}, map[string]string{"internal": "/0", "extern1": "/1", "extern2": "/2"}},
		{"none", map[string]bool{}, map[string]string{}},
	}

	for _, tc := range cases {
		got := ResolveChannels("elsys", tc.matched)
		is.Equal(len(got), len(tc.want))
		for role, want := range tc.want {
			is.Equal(got[role], want)
		}
	}
}

func TestElsysSHT3XChannels(t *testing.T) {
	is := is.New(t)

	// Humidity without pulse keeps no suffix.
	got := ResolveChannels("elsys-sht3x", map[string]bool{"humidity": true})
	is.Equal(got["humidity"], "")

	// Humidity with pulse takes /0 and pulse takes /1.
	got = ResolveChannels("elsys-sht3x", map[string]bool{"humidity": true, "pulse": true})
	is.Equal(got["humidity"], "/0")
	is.Equal(got["pulse"], "/1")

	// Pulse alone still takes /1.
	got = ResolveChannels("elsys-sht3x", map[string]bool{"pulse": true})
	is.Equal(got["pulse"], "/1")

	// Temperature logic is shared with the plain strategy.
	got = ResolveChannels("elsys-sht3x", map[string]bool{"internal": true, "extern1": true})
	is.Equal(got["internal"], "/0")
	is.Equal(got["extern1"], "/1")
}

func TestChannelStrategyValidation(t *testing.T) {
	is := is.New(t)

	groups := map[string]Group{
		"temp": {Name: "temp", URN: "urn:oma:lwm2m:ext:3303", Role: "internal", ChannelStrategy: "elsys"},
	}

	// Unknown strategy fails.
	bad := map[string]Group{
		"temp": {Name: "temp", URN: "urn:oma:lwm2m:ext:3303", Role: "internal", ChannelStrategy: "nope"},
	}
	rules := []Rule{{ID: "t", From: "temperature", Group: "temp", Resource: "5700"}}
	is.True(ValidateRules(rules, bad) != nil)

	// Unknown role fails.
	bad = map[string]Group{
		"temp": {Name: "temp", URN: "urn:oma:lwm2m:ext:3303", Role: "nope", ChannelStrategy: "elsys"},
	}
	is.True(ValidateRules(rules, bad) != nil)

	// Strategy with a static channel fails.
	bad = map[string]Group{
		"temp": {Name: "temp", URN: "urn:oma:lwm2m:ext:3303", Role: "internal", Channel: "/0", ChannelStrategy: "elsys"},
	}
	is.True(ValidateRules(rules, bad) != nil)

	// Role without strategy fails.
	bad = map[string]Group{
		"temp": {Name: "temp", URN: "urn:oma:lwm2m:ext:3303", Role: "internal"},
	}
	is.True(ValidateRules(rules, bad) != nil)

	// Strategy with a governed role passes.
	is.NoErr(ValidateRules(rules, groups))
}
