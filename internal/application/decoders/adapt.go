package decoders

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/application/types"
)

// GenericDecoderFunc decodes an event to a generic result.
// Adapters own legacy gating, errors and normalization.
type GenericDecoderFunc func(ctx context.Context, e types.Event) (types.DecodedResult, error)

// AdaptLegacy wraps a typed legacy decoder. It marshals the typed
// payload to generic maps and copies status verbatim. Sensor fixups
// live in per-sensor adapters, never here.
func AdaptLegacy(fn DecoderFunc) GenericDecoderFunc {
	return func(ctx context.Context, e types.Event) (types.DecodedResult, error) {
		p, err := fn(ctx, e)
		if err != nil {
			return types.DecodedResult{}, err
		}
		data, err := marshalPayload(p)
		if err != nil {
			return types.DecodedResult{}, err
		}
		code, messages := p.Error()
		return types.DecodedResult{
			Data:     data,
			Mapping:  data,
			Battery:  p.BatteryLevel(),
			Code:     code,
			Messages: messages,
		}, nil
	}
}

// marshalPayload converts a typed payload to a generic map through
// JSON. Marshal errors fail explicitly, never silently.
func marshalPayload(p types.SensorPayload) (types.GenericPayload, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	m := types.GenericPayload{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return m, nil
}
