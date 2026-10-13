package decoders

import (
	"context"
	"errors"
	"testing"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/matryer/is"
)

type adaptFixture struct {
	Temp    *float64 `json:"temperature,omitempty"`
	Battery *int     `json:"battery,omitempty"`
}

func (a adaptFixture) BatteryLevel() *int { return a.Battery }
func (a adaptFixture) Error() (string, []string) {
	if a.Temp == nil {
		return "E1", []string{"no temp"}
	}
	return "", nil
}

func TestAdaptLegacy(t *testing.T) {
	is := is.New(t)

	temp := 21.5
	bat := 87
	fn := AdaptLegacy(func(ctx context.Context, e types.Event) (types.SensorPayload, error) {
		return adaptFixture{Temp: &temp, Battery: &bat}, nil
	})

	res, err := fn(t.Context(), types.Event{})
	is.NoErr(err)
	is.Equal(res.Data["temperature"], 21.5)
	is.Equal(*res.Battery, 87)
	code, messages := res.Error()
	is.Equal(code, "")
	is.Equal(len(messages), 0)

	// Data and Mapping share the payload when no adaptation applies.
	res.Data["temperature"] = 0.0
	is.Equal(res.Mapping["temperature"], 0.0)
}

func TestAdaptLegacyError(t *testing.T) {
	is := is.New(t)

	sentinel := errors.New("sentinel")
	fn := AdaptLegacy(func(ctx context.Context, e types.Event) (types.SensorPayload, error) {
		return nil, sentinel
	})

	_, err := fn(t.Context(), types.Event{})
	is.True(errors.Is(err, sentinel))
}

func TestAdaptLegacyErrorStatus(t *testing.T) {
	is := is.New(t)

	fn := AdaptLegacy(func(ctx context.Context, e types.Event) (types.SensorPayload, error) {
		return adaptFixture{}, nil
	})

	res, err := fn(t.Context(), types.Event{})
	is.NoErr(err)
	code, messages := res.Error()
	is.Equal(code, "E1")
	is.Equal(messages, []string{"no temp"})
}
