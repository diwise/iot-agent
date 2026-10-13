package js

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/dop251/goja"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Entry styles select the script entrypoint.
const (
	EntryTTN        = "ttn-decodeUplink"
	EntryChirpStack = "chirpstack-Decode"
)

// DefaultTimeout bounds one decode when the caller sets no deadline.
// It covers queue wait, script init, function call and export.
const DefaultTimeout = 5 * time.Second

// DefaultMaxConcurrent bounds simultaneous JS executions per decoder.
const DefaultMaxConcurrent = 16

// DefaultMaxQueued bounds waiting decodes per decoder. Excess load
// fails fast with ErrTooBusy instead of growing an unbounded queue.
const DefaultMaxQueued = 64

// Export and input budgets. All sizes are deliberately generous for
// LoRa-sized uplinks; they bound abuse, not normal decoders.
const (
	// MaxScriptBytes bounds the script source at compile time.
	MaxScriptBytes = 256 << 10
	// MaxInputBytes bounds one uplink payload.
	MaxInputBytes = 16 << 10
	// MaxResultFields bounds total object fields, recursively.
	MaxResultFields = 1024
	// MaxExportDepth bounds nesting depth (maps and lists).
	MaxExportDepth = 10
	// MaxListItems bounds one exported list.
	MaxListItems = 256
	// MaxStringLen bounds one exported string.
	MaxStringLen = 4 << 10
	// MaxTTNErrors bounds TTN error entries.
	MaxTTNErrors = 32
	// MaxTTNErrorChars bounds joined TTN error text.
	MaxTTNErrorChars = 500
	// MaxWarnings bounds TTN warnings.
	MaxWarnings = 32
)

// ErrTooBusy reports a full JS wait queue.
var ErrTooBusy = errors.New("js decoder queue is full")

// JSDecoder runs one external decoder script. The compiled program is
// shared. Every execution uses a fresh runtime, never shared between
// goroutines. Runtimes expose no file, process or network APIs.
type JSDecoder struct {
	SensorType    string
	ScriptRef     string
	EntryStyle    string
	Program       *goja.Program
	Timeout       time.Duration
	MaxConcurrent int
	MaxQueued     int

	sem    chan struct{}
	queued atomic.Int64
}

// JSInput is one decode call.
type JSInput struct {
	Bytes    []byte
	FPort    int
	RecvTime time.Time
}

// Compile compiles a script once and probes the selected entrypoint
// in a throwaway runtime under budget. Uncompilable scripts and
// missing entrypoints fail here, never at decode time.
func Compile(sensorType, scriptRef, script, entryStyle string) (*JSDecoder, error) {
	if entryStyle != EntryTTN && entryStyle != EntryChirpStack {
		return nil, fmt.Errorf("sensor %s script %s: unknown entry style %q", sensorType, scriptRef, entryStyle)
	}
	if len(script) > MaxScriptBytes {
		return nil, fmt.Errorf("sensor %s script %s: script has %d bytes, limit is %d", sensorType, scriptRef, len(script), MaxScriptBytes)
	}

	program, err := goja.Compile(scriptRef, script, true)
	if err != nil {
		return nil, fmt.Errorf("sensor %s script %s: compile: %w", sensorType, scriptRef, err)
	}

	d := &JSDecoder{
		SensorType:    sensorType,
		ScriptRef:     scriptRef,
		EntryStyle:    entryStyle,
		Program:       program,
		Timeout:       DefaultTimeout,
		MaxConcurrent: DefaultMaxConcurrent,
		MaxQueued:     DefaultMaxQueued,
	}
	d.sem = make(chan struct{}, d.MaxConcurrent)

	if err := d.probeEntry(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *JSDecoder) probeEntry() error {
	vm := goja.New()
	done := make(chan error, 1)
	go func() {
		_, err := vm.RunProgram(d.Program)
		if err != nil {
			done <- err
			return
		}
		_, ok := goja.AssertFunction(vm.Get(d.entryName()))
		if !ok {
			done <- fmt.Errorf("sensor %s script %s: entrypoint %q not found", d.SensorType, d.ScriptRef, d.entryName())
			return
		}
		done <- nil
	}()

	timeout := time.NewTimer(d.Timeout)
	defer timeout.Stop()
	select {
	case err := <-done:
		return err
	case <-timeout.C:
		vm.Interrupt("probe timeout")
		<-done
		return fmt.Errorf("sensor %s script %s: probe timeout", d.SensorType, d.ScriptRef)
	}
}

func (d *JSDecoder) entryName() string {
	if d.EntryStyle == EntryChirpStack {
		return "Decode"
	}
	return "decodeUplink"
}

// Decode runs one uplink through the script and returns the decoded
// fields unchanged with TTN warnings. TTN envelopes classify errors
// as decoder errors. Cancellation and timeout interrupt the runtime.
// The total budget covers queue wait, init, call and export.
func (d *JSDecoder) Decode(ctx context.Context, in JSInput) (types.GenericPayload, []string, error) {
	log := logging.GetFromContext(ctx).With("sensor_type", d.SensorType, "script_ref", d.ScriptRef, "entry_style", d.EntryStyle)
	start := time.Now()

	if in.FPort < 0 || in.FPort > 255 {
		addJSError(ctx, d, "fport")
		return nil, nil, fmt.Errorf("sensor %s script %s: %w: %d", d.SensorType, d.ScriptRef, types.ErrInvalidFPort, in.FPort)
	}
	if len(in.Bytes) > MaxInputBytes {
		addJSError(ctx, d, "input-too-large")
		return nil, nil, fmt.Errorf("sensor %s script %s: input has %d bytes, limit is %d", d.SensorType, d.ScriptRef, len(in.Bytes), MaxInputBytes)
	}

	timeout := d.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		if left := time.Until(deadline); left < timeout {
			timeout = left
		}
	}
	if timeout <= 0 {
		return nil, nil, fmt.Errorf("sensor %s script %s: %w", d.SensorType, d.ScriptRef, context.DeadlineExceeded)
	}
	deadline := start.Add(timeout)
	remaining := func() time.Duration { return time.Until(deadline) }

	// Bounded wait queue: reject overload instead of blocking forever.
	if d.queued.Add(1) > int64(d.MaxQueued) {
		d.queued.Add(-1)
		addJSError(ctx, d, "busy")
		return nil, nil, fmt.Errorf("sensor %s script %s: %w", d.SensorType, d.ScriptRef, ErrTooBusy)
	}
	select {
	case d.sem <- struct{}{}:
		d.queued.Add(-1)
	case <-ctx.Done():
		d.queued.Add(-1)
		return nil, nil, fmt.Errorf("sensor %s script %s: %w", d.SensorType, d.ScriptRef, ctx.Err())
	case <-time.After(remaining()):
		d.queued.Add(-1)
		addJSTimeout(ctx, d)
		return nil, nil, fmt.Errorf("sensor %s script %s: decode timeout waiting for slot", d.SensorType, d.ScriptRef)
	}
	defer func() { <-d.sem }()

	if remaining() <= 0 {
		return nil, nil, fmt.Errorf("sensor %s script %s: %w", d.SensorType, d.ScriptRef, context.DeadlineExceeded)
	}

	vm := goja.New()
	type outcome struct {
		payload  types.GenericPayload
		warnings []string
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		payload, warnings, err := d.run(vm, in)
		done <- outcome{payload, warnings, err}
	}()

	timer := time.NewTimer(remaining())
	defer timer.Stop()
	select {
	case out := <-done:
		recordJSDuration(ctx, d, time.Since(start).Seconds())
		if out.err != nil {
			reason := classifyError(ctx, out.err)
			addJSError(ctx, d, reason)
			log.Debug("js decode failed", "reason", reason, "err", out.err.Error())
			return nil, nil, fmt.Errorf("sensor %s script %s: %w", d.SensorType, d.ScriptRef, out.err)
		}
		return out.payload, out.warnings, nil
	case <-ctx.Done():
		vm.Interrupt("cancelled")
		<-done
		addJSTimeout(ctx, d)
		return nil, nil, fmt.Errorf("sensor %s script %s: %w", d.SensorType, d.ScriptRef, ctx.Err())
	case <-timer.C:
		vm.Interrupt("timeout")
		<-done
		addJSTimeout(ctx, d)
		return nil, nil, fmt.Errorf("sensor %s script %s: decode timeout", d.SensorType, d.ScriptRef)
	}
}

func (d *JSDecoder) run(vm *goja.Runtime, in JSInput) (types.GenericPayload, []string, error) {
	if _, err := vm.RunProgram(d.Program); err != nil {
		return nil, nil, fmt.Errorf("init: %w", err)
	}
	entry, ok := goja.AssertFunction(vm.Get(d.entryName()))
	if !ok {
		return nil, nil, fmt.Errorf("entrypoint %q not found", d.entryName())
	}

	if d.EntryStyle == EntryChirpStack {
		bytes := copyBytes(in.Bytes)
		res, err := entry(goja.Undefined(), vm.ToValue(in.FPort), vm.ToValue(bytes))
		if err != nil {
			return nil, nil, fmt.Errorf("decode: %w", err)
		}
		payload, err := exportPayload(res)
		return payload, nil, err
	}

	obj := vm.NewObject()
	bytes := copyBytes(in.Bytes)
	if err := obj.Set("bytes", vm.ToValue(bytes)); err != nil {
		return nil, nil, err
	}
	if err := obj.Set("fPort", in.FPort); err != nil {
		return nil, nil, err
	}
	recvTime, err := vm.New(vm.Get("Date"), vm.ToValue(in.RecvTime.UnixMilli()))
	if err != nil {
		return nil, nil, fmt.Errorf("recvTime Date: %w", err)
	}
	if err := obj.Set("recvTime", recvTime); err != nil {
		return nil, nil, err
	}
	res, err := entry(goja.Undefined(), obj)
	if err != nil {
		return nil, nil, fmt.Errorf("decode: %w", err)
	}
	return exportEnvelope(res)
}

func copyBytes(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

// exportEnvelope reads a TTN {data, warnings, errors} envelope.
// Non-empty errors are decoder errors. Warnings pass through as
// diagnostics. Data passes through unchanged after budget checks.
func exportEnvelope(res goja.Value) (types.GenericPayload, []string, error) {
	exported := res.Export()
	m, ok := exported.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("envelope must be an object, got %T", exported)
	}
	if err := validateEnvelopeShape(m); err != nil {
		return nil, nil, err
	}
	if rawErrs, ok := m["errors"]; ok && rawErrs != nil {
		errs, ok := rawErrs.([]any)
		if !ok {
			return nil, nil, fmt.Errorf("ttn errors must be a list")
		}
		if len(errs) > 0 {
			if len(errs) > MaxTTNErrors {
				return nil, nil, fmt.Errorf("ttn errors has %d entries, limit is %d", len(errs), MaxTTNErrors)
			}
			msgs := make([]string, 0, len(errs))
			for _, e := range errs {
				s, ok := e.(string)
				if !ok {
					return nil, nil, fmt.Errorf("ttn errors must be strings")
				}
				if len(s) > MaxStringLen {
					return nil, nil, fmt.Errorf("ttn error entry exceeds %d chars", MaxStringLen)
				}
				msgs = append(msgs, s)
			}
			joined := strings.Join(msgs, "; ")
			if len(joined) > MaxTTNErrorChars {
				joined = joined[:MaxTTNErrorChars]
			}
			return nil, nil, fmt.Errorf("ttn errors: %s", joined)
		}
	}
	var warnings []string
	if rawWarn, ok := m["warnings"]; ok && rawWarn != nil {
		warns, ok := rawWarn.([]any)
		if !ok {
			return nil, nil, fmt.Errorf("ttn warnings must be a list")
		}
		if len(warns) > MaxWarnings {
			return nil, nil, fmt.Errorf("ttn warnings has %d entries, limit is %d", len(warns), MaxWarnings)
		}
		for _, w := range warns {
			s, ok := w.(string)
			if !ok {
				return nil, nil, fmt.Errorf("ttn warnings must be strings")
			}
			if len(s) > MaxStringLen {
				return nil, nil, fmt.Errorf("ttn warning exceeds %d chars", MaxStringLen)
			}
			warnings = append(warnings, s)
		}
	}
	raw, ok := m["data"]
	if !ok || raw == nil {
		return nil, nil, fmt.Errorf("envelope has no data")
	}
	payload, err := exportPayloadValue(raw)
	if err != nil {
		return nil, nil, err
	}
	return payload, warnings, nil
}

// exportPayload accepts a flat decoded object.
func exportPayload(res goja.Value) (types.GenericPayload, error) {
	return exportPayloadValue(res.Export())
}

func exportPayloadValue(v any) (types.GenericPayload, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decoded payload must be an object, got %T", v)
	}
	if err := validatePayload(m); err != nil {
		return nil, err
	}
	return m, nil
}

// validateEnvelopeShape bounds the TTN envelope itself before reading
// data. It prevents a huge envelope from bypassing payload limits.
func validateEnvelopeShape(m map[string]any) error {
	if len(m) > MaxResultFields {
		return fmt.Errorf("envelope has %d fields, limit is %d", len(m), MaxResultFields)
	}
	return nil
}

// validatePayload bounds total fields, depth, list sizes, string
// lengths and numeric finiteness, then proves JSON-serializability.
// NaN/Inf in raw data is a decoder/serialization error. Cycles and
// values without JSON representation are rejected.
func validatePayload(m map[string]any) error {
	v := &validator{fields: 0}
	if err := v.walkMap(m, 1); err != nil {
		return err
	}
	if b, err := json.Marshal(m); err != nil {
		return fmt.Errorf("decoded payload is not JSON-serializable: %w", err)
	} else if len(b) > MaxScriptBytes {
		return fmt.Errorf("decoded payload has %d JSON bytes, limit is %d", len(b), MaxScriptBytes)
	}
	return nil
}

type validator struct {
	fields int
}

func (v *validator) walkMap(m map[string]any, depth int) error {
	if depth > MaxExportDepth {
		return fmt.Errorf("decoded payload exceeds depth %d", MaxExportDepth)
	}
	if len(m) > MaxResultFields {
		return fmt.Errorf("decoded payload has %d fields, limit is %d", len(m), MaxResultFields)
	}
	for _, val := range m {
		v.fields++
		if v.fields > MaxResultFields {
			return fmt.Errorf("decoded payload has more than %d total fields", MaxResultFields)
		}
		if err := v.walk(val, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (v *validator) walk(val any, depth int) error {
	switch t := val.(type) {
	case nil, bool, string:
		if s, ok := val.(string); ok && len(s) > MaxStringLen {
			return fmt.Errorf("decoded string exceeds %d chars", MaxStringLen)
		}
		return nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		if f, err := toFloat(val); err == nil {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return fmt.Errorf("decoded payload contains non-finite number")
			}
		}
		_ = t
		return nil
	case map[string]any:
		return v.walkMap(t, depth)
	case []any:
		if depth > MaxExportDepth {
			return fmt.Errorf("decoded payload exceeds depth %d", MaxExportDepth)
		}
		if len(t) > MaxListItems {
			return fmt.Errorf("decoded list has %d items, limit is %d", len(t), MaxListItems)
		}
		for _, e := range t {
			v.fields++
			if v.fields > MaxResultFields {
				return fmt.Errorf("decoded payload has more than %d total fields", MaxResultFields)
			}
			if err := v.walk(e, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("decoded payload has unsupported type %T", val)
	}
}

func toFloat(val any) (float64, error) {
	switch n := val.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint8:
		return float64(n), nil
	case uint16:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	default:
		return 0, fmt.Errorf("not numeric")
	}
}

func classifyError(ctx context.Context, err error) string {
	if ctx.Err() != nil {
		return "cancelled"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "interrupted"):
		return "timeout"
	case strings.Contains(msg, "init:"):
		return "init"
	case strings.Contains(msg, "entrypoint"):
		return "entrypoint"
	case strings.Contains(msg, "ttn errors"):
		return "ttn-errors"
	default:
		return "script"
	}
}

func jsAttrs(d *JSDecoder, reason string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("sensor_type", d.SensorType),
		attribute.String("reason", reason),
	}
}

func addJSError(ctx context.Context, d *JSDecoder, reason string) {
	if jsErrors == nil {
		return
	}
	jsErrors.Add(ctx, 1, metric.WithAttributes(jsAttrs(d, reason)...))
}

func addJSTimeout(ctx context.Context, d *JSDecoder) {
	if jsTimeouts == nil {
		return
	}
	jsTimeouts.Add(ctx, 1, metric.WithAttributes(attribute.String("sensor_type", d.SensorType)))
}

func recordJSDuration(ctx context.Context, d *JSDecoder, seconds float64) {
	if jsDuration == nil {
		return
	}
	jsDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("sensor_type", d.SensorType)))
}

var (
	jsMeters     metric.Meter
	jsErrors     metric.Int64Counter
	jsTimeouts   metric.Int64Counter
	jsDuration   metric.Float64Histogram
	jsMetersOnce sync.Once
)

func initJSMeters() {
	jsMetersOnce.Do(func() {
		jsMeters = otel.Meter("iot-agent/js-decoding")
		var err error
		jsErrors, err = jsMeters.Int64Counter("diwise.js.errors.total", metric.WithUnit("1"), metric.WithDescription("Total number of JS decode errors"))
		if err != nil {
			jsErrors = nil
		}
		jsTimeouts, err = jsMeters.Int64Counter("diwise.js.timeouts.total", metric.WithUnit("1"), metric.WithDescription("Total number of JS decode timeouts"))
		if err != nil {
			jsTimeouts = nil
		}
		jsDuration, err = jsMeters.Float64Histogram("diwise.js.duration", metric.WithUnit("s"), metric.WithDescription("JS decode duration in seconds"))
		if err != nil {
			jsDuration = nil
		}
	})
}

func init() {
	initJSMeters()
}
