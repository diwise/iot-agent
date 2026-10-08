package mapper

import (
	"slices"
)

// ChannelStrategy assigns dynamic channel suffixes to group instances.
// Resolve receives the roles of all matched groups and returns the
// channel per governed role. Roles without an entry keep the static
// group channel.
type ChannelStrategy struct {
	Name  string
	Roles []string
	// Resolve maps matched role to channel suffix such as "/0".
	Resolve func(matched map[string]bool) map[string]string
}

var channelStrategies = map[string]ChannelStrategy{
	"elsys": {
		Name:  "elsys",
		Roles: []string{"internal", "extern1", "extern2"},
		Resolve: func(matched map[string]bool) map[string]string {
			return elsysTemperatureChannels(matched)
		},
	},
	"elsys-sht3x": {
		Name:  "elsys-sht3x",
		Roles: []string{"internal", "extern1", "extern2", "humidity", "pulse"},
		Resolve: func(matched map[string]bool) map[string]string {
			out := elsysTemperatureChannels(matched)
			if matched["humidity"] {
				if matched["pulse"] {
					out["humidity"] = "/0"
				} else {
					out["humidity"] = ""
				}
			}
			if matched["pulse"] {
				out["pulse"] = "/1"
			}
			return out
		},
	},
}

// KnownChannelStrategy reports whether a strategy is registered.
func KnownChannelStrategy(name string) bool {
	_, ok := channelStrategies[name]
	return ok
}

// StrategyRoles lists the roles a strategy governs.
func StrategyRoles(name string) ([]string, bool) {
	s, ok := channelStrategies[name]
	if !ok {
		return nil, false
	}
	return slices.Clone(s.Roles), true
}

// ResolveChannels runs a named strategy over matched roles.
func ResolveChannels(name string, matched map[string]bool) map[string]string {
	s, ok := channelStrategies[name]
	if !ok {
		return nil
	}
	return s.Resolve(matched)
}

// elsysTemperatureChannels reproduces the legacy Elsys suffix table.
// The internal temperature takes "/0" when any external temperature
// is present. Each external temperature takes its number when the
// internal temperature is present. Fields without company keep no
// suffix, even when two external temperatures share one device ID.
func elsysTemperatureChannels(matched map[string]bool) map[string]string {
	out := map[string]string{}
	internal, e1, e2 := matched["internal"], matched["extern1"], matched["extern2"]
	if internal {
		if e1 || e2 {
			out["internal"] = "/0"
		} else {
			out["internal"] = ""
		}
	}
	if e1 {
		if internal {
			out["extern1"] = "/1"
		} else {
			out["extern1"] = ""
		}
	}
	if e2 {
		if internal {
			out["extern2"] = "/2"
		} else {
			out["extern2"] = ""
		}
	}
	return out
}
