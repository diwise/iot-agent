package mapper

import (
	"errors"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

// RuleStatus is the per-rule outcome.
type RuleStatus string

const (
	RuleMatched  RuleStatus = "matched"
	RuleMissing  RuleStatus = "missing"
	RuleDisabled RuleStatus = "disabled"
	RuleInvalid  RuleStatus = "invalid"
)

// Rule-result error codes for invalid rules.
const (
	ErrCodeAmbiguous  = "ambiguous"
	ErrCodePath       = "path"
	ErrCodeType       = "type"
	ErrCodeTransform  = "transform"
	ErrCodeGroup      = "group"
	ErrCodeBuild      = "build"
	ErrCodeConstGroup = "const-group"
)

// RuleResult is the structured outcome of one rule.
type RuleResult struct {
	RuleID    string
	Status    RuleStatus
	Path      string
	Target    string
	Index     int
	ErrorCode string
}

// MappingResult holds built objects and per-rule outcomes.
// Objects follow rule order. Results follow rule order with list
// elements expanded inline.
type MappingResult struct {
	Objects []lwm2m.Lwm2mObject
	Results []RuleResult
}

type pendingValue struct {
	resultPos int
	resource  string
	value     any
}

type instance struct {
	group     Group
	channel   string
	index     int
	header    bool
	failed    bool
	pending   []pendingValue
	hasCreate bool
}

// MapToLWM2M maps decoder fields to LWM2M objects using declarative
// rules. Missing fields skip their rule silently. Lookup, transform
// and coercion failures invalidate only their object group. Groups
// without a matched creator resource produce no object, except
// header_only rules on a truthy source.
func MapToLWM2M(deviceID string, ts time.Time, data types.GenericPayload, groups map[string]Group, rules []Rule) MappingResult {
	var res MappingResult
	instances := map[string]*instance{}
	var order []string

	getInstance := func(g Group, channel string, index int, header bool) *instance {
		key := g.Name + "\x00" + channel + "\x00" + fmt.Sprint(index)
		inst, ok := instances[key]
		if !ok {
			inst = &instance{group: g, channel: channel, index: index, header: header}
			instances[key] = inst
			order = append(order, key)
		}
		return inst
	}

	targetOf := func(g Group, resource string) string {
		if resource == "" {
			return g.URN + " (header)"
		}
		return g.URN + "/" + resource
	}

	for _, r := range rules {
		if !r.IsEnabled() {
			res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleDisabled, Index: -1})
			continue
		}
		g, ok := groups[r.Group]
		if !ok {
			res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Index: -1, ErrorCode: ErrCodeGroup})
			continue
		}
		if _, ok := Catalog()[g.URN]; !ok {
			res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Index: -1, ErrorCode: ErrCodeGroup})
			continue
		}

		if r.HeaderOnly {
			v, found, err := data.Lookup(r.From)
			switch {
			case err != nil:
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Path: r.From, Target: targetOf(g, ""), Index: -1, ErrorCode: ErrCodePath})
			case !found || v == nil:
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMissing, Path: r.From, Target: targetOf(g, ""), Index: -1})
			default:
				b, ok := v.(bool)
				if !ok {
					res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Path: r.From, Target: targetOf(g, ""), Index: -1, ErrorCode: ErrCodeType})
					continue
				}
				if !b {
					res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMissing, Path: r.From, Target: targetOf(g, ""), Index: -1})
					continue
				}
				inst := getInstance(g, g.Channel, -1, true)
				pos := len(res.Results)
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMatched, Path: r.From, Target: targetOf(g, ""), Index: -1})
				inst.pending = append(inst.pending, pendingValue{resultPos: pos})
			}
			continue
		}

		if r.List != "" {
			v, found, err := data.Lookup(r.List)
			switch {
			case err != nil:
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Path: r.List, Target: targetOf(g, r.Resource), Index: -1, ErrorCode: ErrCodePath})
				continue
			case !found || v == nil:
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMissing, Path: r.List, Target: targetOf(g, r.Resource), Index: -1})
				continue
			}
			arr, ok := v.([]any)
			if !ok {
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Path: r.List, Target: targetOf(g, r.Resource), Index: -1, ErrorCode: ErrCodeType})
				continue
			}
			if len(arr) == 0 {
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMissing, Path: r.List, Target: targetOf(g, r.Resource), Index: -1})
				continue
			}
			for i, elem := range arr {
				val, code, ok := applyTransforms(elem, r.Transforms)
				if !ok {
					res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Path: r.List, Target: targetOf(g, r.Resource), Index: i, ErrorCode: code})
					inst := getInstance(g, g.Channel, i, false)
					inst.failed = true
					continue
				}
				inst := getInstance(g, g.Channel, i, false)
				pos := len(res.Results)
				res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMatched, Path: r.List, Target: targetOf(g, r.Resource), Index: i})
				inst.pending = append(inst.pending, pendingValue{resultPos: pos, resource: r.Resource, value: val})
			}
			continue
		}

		var value any
		var path string
		var matched bool
		var code string
		if r.Const != nil {
			value, path, matched = r.Const.V, "(const)", true
		} else {
			value, path, matched, code = lookupWithAliases(data, r.From, r.Aliases)
		}
		if !matched {
			status := RuleMissing
			errCode := ""
			if code != "" {
				status, errCode = RuleInvalid, code
			}
			res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: status, Path: path, Target: targetOf(g, r.Resource), Index: -1, ErrorCode: errCode})
			continue
		}
		val, code, ok := applyTransforms(value, r.Transforms)
		if !ok {
			res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleInvalid, Path: path, Target: targetOf(g, r.Resource), Index: -1, ErrorCode: code})
			inst := getInstance(g, g.Channel, -1, false)
			inst.failed = true
			continue
		}
		inst := getInstance(g, g.Channel, -1, false)
		pos := len(res.Results)
		res.Results = append(res.Results, RuleResult{RuleID: r.ID, Status: RuleMatched, Path: path, Target: targetOf(g, r.Resource), Index: -1})
		inst.pending = append(inst.pending, pendingValue{resultPos: pos, resource: r.Resource, value: val})
	}

	ruleGroup := make(map[string]Group, len(rules))
	for _, r := range rules {
		if g, ok := groups[r.Group]; ok {
			ruleGroup[r.ID] = g
		}
	}
	matchedRoles := map[string]bool{}
	for _, rr := range res.Results {
		if rr.Status == RuleMatched {
			if g, ok := ruleGroup[rr.RuleID]; ok && g.Role != "" {
				matchedRoles[g.Role] = true
			}
		}
	}
	resolved := map[string]map[string]string{}
	for _, g := range groups {
		if g.ChannelStrategy == "" {
			continue
		}
		if _, done := resolved[g.ChannelStrategy]; !done {
			resolved[g.ChannelStrategy] = ResolveChannels(g.ChannelStrategy, matchedRoles)
		}
	}

	for _, key := range order {
		inst := instances[key]
		if inst.failed {
			continue
		}
		channel := inst.channel
		if inst.group.ChannelStrategy != "" {
			if m, ok := resolved[inst.group.ChannelStrategy]; ok {
				if ch, ok := m[inst.group.Role]; ok {
					channel = ch
				} else {
					channel = ""
				}
			}
		}
		objectID := deviceID + channel
		if inst.header {
			obj, err := AssembleObject(inst.group.URN, objectID, ts, map[string]any{})
			if err != nil {
				flipInstance(&res, inst, ErrCodeBuild)
				continue
			}
			res.Objects = append(res.Objects, obj)
			continue
		}
		def := Catalog()[inst.group.URN]
		values := make(map[string]any, len(inst.pending))
		creator := false
		for _, p := range inst.pending {
			kind, ok := def.Resources[p.resource]
			if !ok {
				flipResult(&res, p.resultPos, ErrCodeBuild)
				inst.failed = true
				break
			}
			c, err := coerceValue(kind.Kind, p.value)
			if err != nil {
				flipResult(&res, p.resultPos, ErrCodeType)
				inst.failed = true
				break
			}
			values[p.resource] = c
			if kind.Creator {
				creator = true
			}
		}
		if inst.failed {
			continue
		}
		if !creator {
			continue
		}
		obj, err := AssembleObject(inst.group.URN, objectID, ts, values)
		if err != nil {
			flipInstance(&res, inst, ErrCodeBuild)
			continue
		}
		res.Objects = append(res.Objects, obj)
	}

	return res
}

// lookupWithAliases resolves From then ordered aliases. Explicit null
// stops the chain and counts as missing. It returns the used path and
// a machine-readable code for failures.
func lookupWithAliases(data types.GenericPayload, from string, aliases []string) (value any, path string, matched bool, code string) {
	paths := append([]string{from}, aliases...)
	for _, p := range paths {
		v, found, err := data.Lookup(p)
		if err != nil {
			if errors.Is(err, types.ErrAmbiguousKey) {
				return nil, p, false, ErrCodeAmbiguous
			}
			return nil, p, false, ErrCodePath
		}
		if !found || v == nil {
			if found {
				return nil, p, false, ""
			}
			continue
		}
		return v, p, true, ""
	}
	if len(paths) > 0 {
		return nil, paths[0], false, ""
	}
	return nil, "", false, ""
}

// applyTransforms runs a validated chain. It returns false with a
// machine-readable code on failure.
func applyTransforms(v any, steps []TransformStep) (any, string, bool) {
	cur := v
	for _, t := range steps {
		fn, err := GetTransform(t.Name, t.Params)
		if err != nil {
			return nil, ErrCodeTransform, false
		}
		cur, err = fn(cur)
		if err != nil {
			return nil, ErrCodeTransform, false
		}
	}
	return cur, "", true
}

// coerceValue converts a transformed value to a catalog kind.
func coerceValue(kind ValueKind, v any) (any, error) {
	switch kind {
	case KindFloat:
		return types.AsFloat(v)
	case KindInt:
		return types.AsInt(v)
	case KindBool:
		return types.AsBool(v)
	case KindString:
		if s, ok := v.(string); ok {
			return s, nil
		}
		return nil, fmt.Errorf("expected string, got %T", v)
	default:
		return nil, fmt.Errorf("unknown value kind %d", int(kind))
	}
}

func flipResult(res *MappingResult, pos int, code string) {
	if pos >= 0 && pos < len(res.Results) {
		res.Results[pos].Status = RuleInvalid
		res.Results[pos].ErrorCode = code
	}
}

func flipInstance(res *MappingResult, inst *instance, code string) {
	for _, p := range inst.pending {
		flipResult(res, p.resultPos, code)
	}
}
