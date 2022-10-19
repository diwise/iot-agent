package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/infrastructure/services/mqtt"
)

func PresenceDecoder(ctx context.Context, ue mqtt.UplinkEvent, fn func(context.Context, Payload) error) error {
	p := Payload{
		DevEUI:    ue.DevEui,
		Timestamp: ue.Timestamp.Format(time.RFC3339Nano),
	}

	obj := struct {
		Presence struct {
			Value *bool `json:"value"`
		} `json:"closeProximityAlarm,omitempty"`
	}{}

	err := json.Unmarshal(ue.Object, &obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal presence payload: %s", err.Error())
	}

	if obj.Presence.Value != nil {
		present := struct {
			Presence bool `json:"present"`
		}{
			*obj.Presence.Value,
		}
		p.Measurements = append(p.Measurements, present)
	}

	return fn(ctx, p)
}
