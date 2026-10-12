package mapper

import (
	"fmt"
	"strconv"

	"github.com/diwise/iot-agent/internal/application/types"
)

// StatusIssue is one classified status finding.
type StatusIssue struct {
	Path   string
	Code   string
	Detail string
}

// StatusResult holds device status from explicit mapping.
type StatusResult struct {
	Battery  *int
	Code     string
	Messages []string
	Issues   []StatusIssue
}

// ApplyStatus reads battery, code and messages from mapped sources.
// Empty paths mean absent. Missing and null sources mean absent.
func ApplyStatus(data types.GenericPayload, sm StatusMapping) StatusResult {
	var res StatusResult

	if sm.Battery.Path != "" {
		if b, issues := applyBattery(data, sm.Battery); len(issues) > 0 {
			res.Issues = append(res.Issues, issues...)
		} else {
			res.Battery = b
		}
	}

	if sm.Code.Path != "" {
		v, found, err := data.Lookup(sm.Code.Path)
		switch {
		case err != nil:
			res.Issues = append(res.Issues, StatusIssue{Path: sm.Code.Path, Code: "path", Detail: err.Error()})
		case !found || v == nil:
		default:
			switch c := v.(type) {
			case string:
				res.Code = c
			default:
				if i, ierr := types.AsInt(v); ierr != nil {
					res.Issues = append(res.Issues, StatusIssue{Path: sm.Code.Path, Code: "type", Detail: fmt.Sprintf("expected string, got %T", v)})
				} else {
					res.Code = strconv.Itoa(i)
				}
			}
		}
	}

	if sm.Messages.Path != "" {
		v, found, err := data.Lookup(sm.Messages.Path)
		switch {
		case err != nil:
			res.Issues = append(res.Issues, StatusIssue{Path: sm.Messages.Path, Code: "path", Detail: err.Error()})
		case !found || v == nil:
		default:
			switch m := v.(type) {
			case string:
				res.Messages = []string{m}
			case []any:
				msgs := make([]string, 0, len(m))
				ok := true
				for _, e := range m {
					s, isStr := e.(string)
					if !isStr {
						ok = false
						break
					}
					msgs = append(msgs, s)
				}
				if !ok {
					res.Issues = append(res.Issues, StatusIssue{Path: sm.Messages.Path, Code: "type", Detail: "expected list of strings"})
				} else {
					res.Messages = msgs
				}
			default:
				res.Issues = append(res.Issues, StatusIssue{Path: sm.Messages.Path, Code: "type", Detail: fmt.Sprintf("expected string list, got %T", v)})
			}
		}
	}

	return res
}

func applyBattery(data types.GenericPayload, src StatusSource) (*int, []StatusIssue) {
	v, found, err := data.Lookup(src.Path)
	if err != nil {
		return nil, []StatusIssue{{Path: src.Path, Code: "path", Detail: err.Error()}}
	}
	if !found || v == nil {
		return nil, nil
	}
	for _, t := range src.Transforms {
		fn, ferr := GetTransform(t.Name, t.Params)
		if ferr != nil {
			return nil, []StatusIssue{{Path: src.Path, Code: "transform", Detail: ferr.Error()}}
		}
		v, ferr = fn(v)
		if ferr != nil {
			return nil, []StatusIssue{{Path: src.Path, Code: "transform", Detail: ferr.Error()}}
		}
	}
	if v == nil {
		return nil, nil
	}
	b, berr := types.BatteryPercent(v)
	if berr != nil {
		return nil, []StatusIssue{{Path: src.Path, Code: "battery", Detail: berr.Error()}}
	}
	return b, nil
}
