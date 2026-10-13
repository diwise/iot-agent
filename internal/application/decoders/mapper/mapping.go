// Package mapper maps generic decoder payloads to LWM2M objects using
// declarative rules. It holds no decoder logic and no conditions:
// decoders own all gating, errors and normalization.
package mapper

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/diwise/iot-agent/internal/application/types"
	"gopkg.in/yaml.v3"
)

// TransformStep is one named transform with validated parameters.
// Steps run in declared order.
type TransformStep struct {
	Name   string         `yaml:"name"`
	Params map[string]any `yaml:"params,omitempty"`
}

// ConstValue holds a constant rule source. Constants never act as paths.
type ConstValue struct {
	V any `yaml:"-"`
}

// UnmarshalYAML accepts any scalar, list or map as a constant value.
func (c *ConstValue) UnmarshalYAML(value *yaml.Node) error {
	var v any
	if err := value.Decode(&v); err != nil {
		return err
	}
	c.V = v
	return nil
}

// Rule maps one source to one resource of an object group.
// Exactly one source applies: From with optional Aliases, List, or
// Const. HeaderOnly rules carry From but no resource, transform,
// const or list.
type Rule struct {
	ID         string          `yaml:"id"`
	From       string          `yaml:"from,omitempty"`
	Aliases    []string        `yaml:"aliases,omitempty"`
	Group      string          `yaml:"group"`
	Resource   string          `yaml:"resource,omitempty"`
	Const      *ConstValue     `yaml:"const,omitempty"`
	Transforms []TransformStep `yaml:"transforms,omitempty"`
	Enabled    *bool           `yaml:"enabled,omitempty"`
	HeaderOnly bool            `yaml:"header_only,omitempty"`
	List       string          `yaml:"list,omitempty"`
}

// IsEnabled reports true when Enabled is nil or true.
func (r Rule) IsEnabled() bool {
	return r.Enabled == nil || *r.Enabled
}

// Group names an object type with its channel behavior.
type Group struct {
	// Name comes from the configuration key and has no YAML field.
	Name string `yaml:"-"`
	URN  string `yaml:"urn"`
	// Channel is a static suffix such as "/0". Empty means no suffix.
	// Groups with a ChannelStrategy must leave Channel empty.
	Channel string `yaml:"channel,omitempty"`
	// ChannelStrategy selects dynamic channel assignment.
	// Empty means the static Channel applies.
	ChannelStrategy string `yaml:"strategy,omitempty"`
	// Role names this group's part in a channel strategy,
	// e.g. internal, extern1 or extern2 for Elsys temperatures.
	// A role requires a strategy and must belong to it.
	Role string `yaml:"role,omitempty"`
}

// StatusSource selects one status field by path with an optional
// transform chain. Empty path means absent.
type StatusSource struct {
	Path       string          `yaml:"path,omitempty"`
	Transforms []TransformStep `yaml:"transforms,omitempty"`
}

// StatusMapping selects battery, code and messages for device status.
// Empty sources mean absent.
type StatusMapping struct {
	Battery  StatusSource `yaml:"battery,omitempty"`
	Code     StatusSource `yaml:"code,omitempty"`
	Messages StatusSource `yaml:"messages,omitempty"`
}

var channelPattern = regexp.MustCompile(`^(/[0-9]+)?$`)

// ValidateRules checks rules and groups without decoder data.
// Unknown groups, URNs, resources, transforms and bad rule shapes
// fail here, never at mapping time.
func ValidateRules(rules []Rule, groups map[string]Group) error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	seenID := map[string]bool{}
	seenTarget := map[string]string{}
	headerGroups := map[string]string{}

	for _, r := range rules {
		if r.ID == "" {
			fail("rule without ID")
			continue
		}
		if seenID[r.ID] {
			fail("rule %q: duplicate ID", r.ID)
			continue
		}
		seenID[r.ID] = true

		g, ok := groups[r.Group]
		if !ok {
			fail("rule %q: unknown group %q", r.ID, r.Group)
			continue
		}
		def, ok := Catalog()[g.URN]
		if !ok {
			fail("rule %q: group %q uses unknown URN %q", r.ID, r.Group, g.URN)
			continue
		}
		if g.ChannelStrategy != "" {
			if !KnownChannelStrategy(g.ChannelStrategy) {
				fail("rule %q: group %q uses unknown channel strategy %q", r.ID, r.Group, g.ChannelStrategy)
			} else {
				roles, _ := StrategyRoles(g.ChannelStrategy)
				if g.Role == "" {
					fail("rule %q: group %q needs a role for strategy %q", r.ID, r.Group, g.ChannelStrategy)
				} else if !slices.Contains(roles, g.Role) {
					fail("rule %q: group %q role %q is not in strategy %q", r.ID, r.Group, g.Role, g.ChannelStrategy)
				}
			}
			if g.Channel != "" {
				fail("rule %q: group %q with a strategy must leave Channel empty", r.ID, r.Group)
			}
		} else if g.Role != "" {
			fail("rule %q: group %q role %q needs a strategy", r.ID, r.Group, g.Role)
		}
		if !channelPattern.MatchString(g.Channel) {
			fail("rule %q: group %q has bad channel %q", r.ID, r.Group, g.Channel)
		}

		if r.HeaderOnly {
			if !isDeviceURN(g.URN) {
				fail("rule %q: header_only is only allowed for the Device group", r.ID)
			}
			if r.Resource != "" {
				fail("rule %q: header_only takes no resource", r.ID)
			}
			if r.From == "" {
				fail("rule %q: header_only needs a source path", r.ID)
			}
			if err := types.ValidatePath(r.From); err != nil {
				fail("rule %q: %v", r.ID, err)
			}
			if r.Const != nil || r.List != "" || len(r.Transforms) > 0 || len(r.Aliases) > 0 {
				fail("rule %q: header_only takes no const, list, transforms or aliases", r.ID)
			}
			if prev, dup := headerGroups[r.Group]; dup {
				fail("rule %q: group %q already has header_only rule %q", r.ID, r.Group, prev)
			} else {
				headerGroups[r.Group] = r.ID
			}
			continue
		}

		hasFrom := r.From != ""
		hasConst := r.Const != nil
		hasList := r.List != ""
		sources := 0
		if hasFrom {
			sources++
		}
		if hasConst {
			sources++
		}
		if hasList {
			sources++
		}
		if sources != 1 {
			fail("rule %q: needs exactly one of from, const or list", r.ID)
			continue
		}

		if r.Resource == "" {
			fail("rule %q: needs a resource", r.ID)
			continue
		}
		if _, ok := def.Resources[r.Resource]; !ok {
			fail("rule %q: group %q (URN %s) has no resource %q", r.ID, r.Group, g.URN, r.Resource)
			continue
		}

		if hasFrom {
			if err := types.ValidatePath(r.From); err != nil {
				fail("rule %q: %v", r.ID, err)
			}
			for _, a := range r.Aliases {
				if err := types.ValidatePath(a); err != nil {
					fail("rule %q: alias: %v", r.ID, err)
				}
			}
		}
		if hasList {
			if err := types.ValidatePath(r.List); err != nil {
				fail("rule %q: list: %v", r.ID, err)
			}
			if len(r.Aliases) > 0 {
				fail("rule %q: list takes no aliases", r.ID)
			}
		}
		if hasConst && r.Const.V == nil {
			fail("rule %q: const null is not allowed", r.ID)
		}

		for _, t := range r.Transforms {
			if _, err := GetTransform(t.Name, t.Params); err != nil {
				fail("rule %q: %v", r.ID, err)
			}
		}

		key := r.Group + "\x00" + r.Resource
		if prev, dup := seenTarget[key]; dup {
			fail("rule %q: duplicate target of rule %q (group %q resource %q)", r.ID, prev, r.Group, r.Resource)
		} else {
			seenTarget[key] = r.ID
		}
	}

	return errors.Join(errs...)
}

func isDeviceURN(urn string) bool {
	return strings.HasSuffix(urn, ":3")
}
