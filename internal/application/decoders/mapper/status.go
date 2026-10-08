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

// ApplyStatus reads battery, code and messages from mapped paths.
// Empty paths mean absent. Missing and null sources mean absent.
// Battery values pass an optional transform chain and strict
// 0-100 validation. Non-string codes accept integral numbers.
// Messages accept a string list or a single string.
func ApplyStatus(data types.GenericPayload, sm StatusMapping) StatusResult {
	var res StatusResult

	if sm.BatteryPath != "" {
		v, found, err := data.Lookup(sm.BatteryPath)
		switch {
		case err != nil:
			res.Issues = append(res.Issues, StatusIssue{Path: sm.BatteryPath, Code: "path", Detail: err.Error()})
		case !found || v == nil:
		default:
			for _, t := range sm.BatteryTransforms {
				fn, ferr := GetTransform(t.Name, t.Params)
				if ferr != nil {
					res.Issues = append(res.Issues, StatusIssue{Path: sm.BatteryPath, Code: "transform", Detail: ferr.Error()})
					v = nil
					break
				}
				v, ferr = fn(v)
				if ferr != nil {
					res.Issues = append(res.Issues, StatusIssue{Path: sm.BatteryPath, Code: "transform", Detail: ferr.Error()})
					v = nil
					break
				}
			}
			if v != nil {
				if b, berr := types.BatteryPercent(v); berr != nil {
					res.Issues = append(res.Issues, StatusIssue{Path: sm.BatteryPath, Code: "battery", Detail: berr.Error()})
				} else {
					res.Battery = b
				}
			}
		}
	}

	if sm.CodePath != "" {
		v, found, err := data.Lookup(sm.CodePath)
		switch {
		case err != nil:
			res.Issues = append(res.Issues, StatusIssue{Path: sm.CodePath, Code: "path", Detail: err.Error()})
		case !found || v == nil:
		default:
			switch c := v.(type) {
			case string:
				res.Code = c
			default:
				if i, ierr := types.AsInt(v); ierr != nil {
					res.Issues = append(res.Issues, StatusIssue{Path: sm.CodePath, Code: "type", Detail: fmt.Sprintf("expected string, got %T", v)})
				} else {
					res.Code = strconv.Itoa(i)
				}
			}
		}
	}

	if sm.MessagePath != "" {
		v, found, err := data.Lookup(sm.MessagePath)
		switch {
		case err != nil:
			res.Issues = append(res.Issues, StatusIssue{Path: sm.MessagePath, Code: "path", Detail: err.Error()})
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
					res.Issues = append(res.Issues, StatusIssue{Path: sm.MessagePath, Code: "type", Detail: "expected list of strings"})
				} else {
					res.Messages = msgs
				}
			default:
				res.Issues = append(res.Issues, StatusIssue{Path: sm.MessagePath, Code: "type", Detail: fmt.Sprintf("expected string list, got %T", v)})
			}
		}
	}

	return res
}
