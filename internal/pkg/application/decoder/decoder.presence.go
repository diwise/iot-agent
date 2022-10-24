package decoder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func PresenceDecoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	obj := struct {
		Presence struct {
			Value *bool `json:"value"`
		} `json:"closeProximityAlarm,omitempty"`
	}{}

	err := json.Unmarshal(ue.Object, &obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal presence payload: %s", err.Error())
	}

	var decorators []payload.PayloadDecoratorFunc
	if obj.Presence.Value != nil {
		decorators = append(decorators, payload.Presence(*obj.Presence.Value))
	}

	if p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...); err == nil {
		return fn(ctx, p)
	} else {
		return err
	}
}
