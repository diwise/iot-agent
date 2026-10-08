package mapper

import (
	"strings"
	"testing"

	"github.com/matryer/is"
)

func testGroups() map[string]Group {
	return map[string]Group{
		"temp":    {Name: "temp", URN: "urn:oma:lwm2m:ext:3303"},
		"device":  {Name: "device", URN: "urn:oma:lwm2m:ext:3"},
		"checkin": {Name: "checkin", URN: "urn:oma:lwm2m:ext:3"},
		"cond":    {Name: "cond", URN: "urn:oma:lwm2m:ext:3327"},
	}
}

func TestValidateRulesOK(t *testing.T) {
	is := is.New(t)

	enabled := true
	rules := []Rule{
		{ID: "t", From: "temperature", Group: "temp", Resource: "5700"},
		{ID: "b", From: "battery", Group: "device", Resource: "9", Enabled: &enabled},
		{ID: "c", From: "check_in", Group: "checkin", HeaderOnly: true},
		{ID: "r0", List: "resistances[0]", Group: "cond", Resource: "5700", Transforms: []TransformStep{{Name: "reciprocal"}}},
		{ID: "k", Group: "temp", Resource: "5750", Const: &ConstValue{V: "air"}},
	}
	is.NoErr(ValidateRules(rules, testGroups()))
}

func TestValidateRulesRejects(t *testing.T) {
	is := is.New(t)

	groups := testGroups()
	cases := []struct {
		name  string
		rules []Rule
		want  string
	}{
		{"empty ID", []Rule{{From: "a", Group: "temp", Resource: "5700"}}, "without ID"},
		{"duplicate ID", []Rule{
			{ID: "x", From: "a", Group: "temp", Resource: "5700"},
			{ID: "x", From: "b", Group: "temp", Resource: "5601"},
		}, "duplicate ID"},
		{"unknown group", []Rule{{ID: "x", From: "a", Group: "nope", Resource: "5700"}}, "unknown group"},
		{"unknown resource", []Rule{{ID: "x", From: "a", Group: "temp", Resource: "9999"}}, "no resource"},
		{"unknown transform", []Rule{{ID: "x", From: "a", Group: "temp", Resource: "5700", Transforms: []TransformStep{{Name: "nope"}}}}, "unknown transform"},
		{"bad transform params", []Rule{{ID: "x", From: "a", Group: "temp", Resource: "5700", Transforms: []TransformStep{{Name: "scale", Params: map[string]any{"factor": "x"}}}}}, "not numeric"},
		{"no source", []Rule{{ID: "x", Group: "temp", Resource: "5700"}}, "exactly one"},
		{"from and const", []Rule{{ID: "x", From: "a", Group: "temp", Resource: "5700", Const: &ConstValue{V: 1.0}}}, "exactly one"},
		{"no resource", []Rule{{ID: "x", From: "a", Group: "temp"}}, "needs a resource"},
		{"const null", []Rule{{ID: "x", Group: "temp", Resource: "5700", Const: &ConstValue{}}}, "const null"},
		{"list with alias", []Rule{{ID: "x", List: "a", Aliases: []string{"b"}, Group: "cond", Resource: "5700"}}, "no aliases"},
		{"bad list path", []Rule{{ID: "x", List: "a[", Group: "cond", Resource: "5700"}}, "unclosed index"},
		{"duplicate target", []Rule{
			{ID: "x", From: "a", Group: "temp", Resource: "5700"},
			{ID: "y", From: "b", Group: "temp", Resource: "5700"},
		}, "duplicate target"},
		{"header with resource", []Rule{{ID: "x", From: "a", Group: "checkin", Resource: "9", HeaderOnly: true}}, "no resource"},
		{"header without source", []Rule{{ID: "x", Group: "checkin", HeaderOnly: true}}, "needs a source"},
		{"header with transform", []Rule{{ID: "x", From: "a", Group: "checkin", HeaderOnly: true, Transforms: []TransformStep{{Name: "identity"}}}}, "no const, list, transforms"},
		{"second header", []Rule{
			{ID: "x", From: "a", Group: "checkin", HeaderOnly: true},
			{ID: "y", From: "b", Group: "checkin", HeaderOnly: true},
		}, "already has header_only"},
		{"header outside device", []Rule{{ID: "x", From: "a", Group: "temp", HeaderOnly: true}}, "only allowed for the Device"},
		{"bad channel", []Rule{{ID: "x", From: "a", Group: "tempchan", Resource: "5700"}}, "bad channel"},
	}

	groups["tempchan"] = Group{Name: "tempchan", URN: "urn:oma:lwm2m:ext:3303", Channel: "zero"}

	for _, tc := range cases {
		err := ValidateRules(tc.rules, groups)
		is.True(err != nil)
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q misses %q", tc.name, err.Error(), tc.want)
		}
	}
}

func TestValidateRulesAcceptsDisabledAndConst(t *testing.T) {
	is := is.New(t)

	disabled := false
	rules := []Rule{
		{ID: "x", From: "a", Group: "temp", Resource: "5700", Enabled: &disabled},
	}
	is.NoErr(ValidateRules(rules, testGroups()))

	// Disabled rules are still validated structurally.
	bad := []Rule{
		{ID: "x", From: "a", Group: "temp", Resource: "5700", Enabled: &disabled, Transforms: []TransformStep{{Name: "nope"}}},
	}
	is.True(ValidateRules(bad, testGroups()) != nil)
}
