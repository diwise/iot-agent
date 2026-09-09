package application

import (
	"context"
	"log/slog"
	"testing"

	"github.com/diwise/iot-agent/internal/application/facades"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

// captureHandler is a minimal slog.Handler that collects log records
// for assertions. Groups are flattened; preformatted attributes are
// attached to every captured record.
type captureHandler struct {
	records *[]slog.Record
	attrs   []slog.Attr
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	r.AddAttrs(h.attrs...)
	*h.records = append(*h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &captureHandler{records: h.records, attrs: merged}
}

func (h *captureHandler) WithGroup(string) slog.Handler {
	return &captureHandler{records: h.records, attrs: h.attrs}
}

// AGENT-004: locks that the device-status publish log carries scalar
// metadata only, never the full status struct or sensor payload.
func TestDeviceStatusPublishLogHasScalarFieldsOnly(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	var records []slog.Record
	ctx = logging.NewContextWithLogger(ctx, slog.New(&captureHandler{records: &records}))

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{})
	ue, err := facades.New("netmore")(ctx, "payload", []byte(senlabT))
	is.NoErr(err)

	is.NoErr(agent.HandleSensorEvent(ctx, ue))

	var found *slog.Record
	for i := range records {
		if records[i].Message == "publish device-status message" {
			found = &records[i]
			break
		}
	}
	is.True(found != nil)

	keys := map[string]bool{}
	found.Attrs(func(a slog.Attr) bool {
		keys[a.Key] = true
		return true
	})

	is.True(keys["device_id"])
	is.True(keys["status_code"])
	is.True(keys["status_messages"])
	is.True(!keys["status"])
}
