package decoders

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/diwise/iot-agent/internal/application/decoders/airquality"
	"github.com/diwise/iot-agent/internal/application/decoders/axsensor"
	"github.com/diwise/iot-agent/internal/application/decoders/elsys"
	"github.com/diwise/iot-agent/internal/application/decoders/enviot"
	"github.com/diwise/iot-agent/internal/application/decoders/js"
	"github.com/diwise/iot-agent/internal/application/decoders/mapper"
	"github.com/diwise/iot-agent/internal/application/decoders/milesight"
	"github.com/diwise/iot-agent/internal/application/decoders/niab"
	"github.com/diwise/iot-agent/internal/application/decoders/qalcosonic"
	"github.com/diwise/iot-agent/internal/application/decoders/senlabt"
	"github.com/diwise/iot-agent/internal/application/decoders/sensative"
	"github.com/diwise/iot-agent/internal/application/decoders/sensefarm"
	"github.com/diwise/iot-agent/internal/application/decoders/talkpool"
	"github.com/diwise/iot-agent/internal/application/decoders/vegapuls"
	"github.com/diwise/iot-agent/internal/application/decoders/x2climate"
	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type DecoderFunc func(ctx context.Context, e types.Event) (types.SensorPayload, error)
type ConverterFunc func(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error)

// SnapshotKind selects the decode path.
type SnapshotKind string

const (
	KindGo     SnapshotKind = "go"
	KindJS     SnapshotKind = "js"
	KindLegacy SnapshotKind = "legacy"
)

// InputSource selects the event field a decoder reads.
type InputSource string

const (
	InputBytes  InputSource = "bytes"
	InputObject InputSource = "object"
)

// Snapshot is one immutable decoder configuration. The same snapshot
// serves a whole uplink from decode to status.
type Snapshot struct {
	ID     string
	Kind   SnapshotKind
	Input  InputSource
	Go     GenericDecoderFunc
	JS     *js.JSDecoder
	Legacy LegacyPair

	Groups map[string]mapper.Group
	Rules  []mapper.Rule
	Status mapper.StatusMapping
	Hash   string
}

// LegacyPair is the explicit Qalcosonic exception and the temporary
// fallback for unmigrated decoder IDs.
type LegacyPair struct {
	Decoder   DecoderFunc
	Converter ConverterFunc
}

// Registry resolves configuration IDs to snapshots.
type Registry interface {
	Get(ctx context.Context, sensorType string) (*Snapshot, bool)
}

type registryImpl struct {
	snapshots map[string]*Snapshot
}

var legacyDecoders = map[string]DecoderFunc{
	"airquality": airquality.Decoder,
	"axsensor":   axsensor.Decoder,
	"enviot":     enviot.Decoder,
	"niab-fls":   niab.Decoder,

	"qalcosonic":     qalcosonic.Decoder,
	"qalcosonic/w1t": qalcosonic.DecoderW1t,
	"qalcosonic/w1h": qalcosonic.DecoderW1h,
	"qalcosonic/w1e": qalcosonic.DecoderW1e,

	"vegapuls_air_41": vegapuls.Decoder,

	"elt_2_hp":        elsys.Decoder,
	"elsys":           elsys.Decoder,
	"elsys/elt/sht3x": elsys.Decoder,
	"elsys_codec":     elsys.Decoder, // deprecated, use elsys

	"milesight":       milesight.Decoder,
	"milesight_am100": milesight.Decoder, // deprecated, use milesight

	"senlabt":      senlabt.Decoder,
	"tem_lab_14ns": senlabt.Decoder, // deprecated, use senlabt

	"sensefarm": sensefarm.Decoder,
	"cube02":    sensefarm.Decoder, // deprecated, use sensefarm

	"sensative":        sensative.Decoder,
	"strips_lora_ms_h": sensative.Decoder, // deprecated, use sensative
	"presence":         sensative.Decoder, // deprecated, use sensative

	"talkpool/oy1210": talkpool.DecoderOy1210,

	"x2climate": x2climate.DecoderX2Climate,
}

var legacyConverters = map[string]ConverterFunc{
	"airquality": airquality.Converter,
	"axsensor":   axsensor.Converter,
	"enviot":     enviot.Converter,
	"niab-fls":   niab.Converter,

	"qalcosonic":     qalcosonic.Converter,
	"qalcosonic/w1t": qalcosonic.Converter,
	"qalcosonic/w1h": qalcosonic.Converter,
	"qalcosonic/w1e": qalcosonic.Converter,

	"vegapuls_air_41": vegapuls.Converter,

	"elt_2_hp":        elsys.Converter,
	"elsys":           elsys.Converter,
	"elsys/elt/sht3x": elsys.ConverterEltSht3x,
	"elsys_codec":     elsys.Converter, // deprecated, use elsys

	"milesight":       milesight.Converter,
	"milesight_am100": milesight.Converter, // deprecated, use milesight

	"senlabt":      senlabt.Converter,
	"tem_lab_14ns": senlabt.Converter, // deprecated, use senlabt

	"sensefarm": sensefarm.Converter,
	"cube02":    sensefarm.Converter, // deprecated, use sensefarm

	"sensative":        sensative.Converter,
	"strips_lora_ms_h": sensative.Converter, // deprecated, use sensative
	"presence":         sensative.Converter, // deprecated, use sensative

	"talkpool/oy1210": talkpool.ConverterOy1210,

	"x2climate": x2climate.ConverterX2Climate,
}

// objectCapable lists Go decoders that read Payload.Object.
// Only these accept input object.
var objectCapable = map[string]bool{
	"enviot":   true,
	"niab-fls": true,
}

func knownGoRef(ref string) bool {
	_, ok := legacyDecoders[strings.ToLower(ref)]
	return ok
}

func isQalcosonicRef(ref string) bool {
	return strings.HasPrefix(strings.ToLower(ref), "qalcosonic")
}

func isQalcosonicID(id string) bool {
	return strings.HasPrefix(strings.ToLower(id), "qalcosonic")
}

// NewRegistry builds legacy snapshots for all known IDs.
// Behavior matches the previous decoder/converter registry.
func NewRegistry() Registry {
	snapshots := make(map[string]*Snapshot, len(legacyDecoders))
	for id, fn := range legacyDecoders {
		snapshots[id] = &Snapshot{
			ID:    id,
			Kind:  KindLegacy,
			Input: InputBytes,
			Legacy: LegacyPair{
				Decoder:   withDecodeMetrics(id, fn),
				Converter: legacyConverters[id],
			},
		}
	}
	return &registryImpl{snapshots: snapshots}
}

// RequiredIDs lists every decoder ID that must have a generic YAML
// entry. Qalcosonic IDs are the explicit legacy exception and are
// never required here.
func RequiredIDs() []string {
	var out []string
	for id := range legacyDecoders {
		if !isQalcosonicID(id) {
			out = append(out, id)
		}
	}
	return out
}

// BuildRegistry builds snapshots from a mappings file. Qalcosonic IDs
// take no YAML entries and always use the legacy path. Every other
// known ID must have a generic entry; missing entries fail startup
// instead of silently falling back to legacy.
func BuildRegistry(ctx context.Context, path string) (Registry, error) {
	cfgs, err := LoadMappings(path)
	if err != nil {
		return nil, err
	}

	snapshots := make(map[string]*Snapshot, len(cfgs)+len(legacyDecoders))
	for id, cfg := range cfgs {
		lowered := strings.ToLower(id)
		if _, dup := snapshots[lowered]; dup {
			return nil, fmt.Errorf("decoder %q collides with another entry after lowercasing", id)
		}
		if isQalcosonicID(id) {
			return nil, fmt.Errorf("decoder %q: qalcosonic stays on the legacy path and takes no YAML entry", id)
		}
		snap, err := buildSnapshot(ctx, id, cfg, filepathDir(path))
		if err != nil {
			return nil, err
		}
		snapshots[lowered] = snap
	}

	var missing []string
	for id := range legacyDecoders {
		if isQalcosonicID(id) {
			continue
		}
		if _, ok := snapshots[strings.ToLower(id)]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("mappings %s is missing generic configuration for %d decoder(s): %s", path, len(missing), strings.Join(missing, ", "))
	}

	for id, fn := range legacyDecoders {
		if !isQalcosonicID(id) {
			continue
		}
		lowered := strings.ToLower(id)
		if _, ok := snapshots[lowered]; ok {
			continue
		}
		snapshots[lowered] = &Snapshot{
			ID:    id,
			Kind:  KindLegacy,
			Input: InputBytes,
			Legacy: LegacyPair{
				Decoder:   withDecodeMetrics(id, fn),
				Converter: legacyConverters[id],
			},
		}
	}

	return &registryImpl{snapshots: snapshots}, nil
}

func buildSnapshot(ctx context.Context, id string, cfg DecoderConfig, dir string) (*Snapshot, error) {
	snap := &Snapshot{
		ID:     id,
		Groups: cfg.Groups,
		Rules:  cfg.Rules,
		Status: cfg.Status,
	}
	switch cfg.InputOrDefault() {
	case "object":
		snap.Input = InputObject
	default:
		snap.Input = InputBytes
	}

	if cfg.Script != "" {
		if snap.Input != InputBytes {
			return nil, fmt.Errorf("decoder %q: scripts support only bytes input", id)
		}
		scriptPath, err := scriptPath(dir, cfg.Script)
		if err != nil {
			return nil, fmt.Errorf("decoder %q: %w", id, err)
		}
		src, err := os.ReadFile(scriptPath)
		if err != nil {
			return nil, fmt.Errorf("decoder %q: read script: %w", id, err)
		}
		decoder, err := js.Compile(id, cfg.Script, string(src), cfg.Entry)
		if err != nil {
			return nil, fmt.Errorf("decoder %q: %w", id, err)
		}
		snap.Kind = KindJS
		snap.JS = decoder
		snap.Hash = configHash(id, cfg, src)
		return snap, nil
	}

	if isQalcosonicRef(cfg.Go) {
		return nil, fmt.Errorf("decoder %q: qalcosonic stays on the legacy path", id)
	}
	if snap.Input == InputObject && !objectCapable[strings.ToLower(cfg.Go)] {
		return nil, fmt.Errorf("decoder %q: go-ref %q does not read object input", id, cfg.Go)
	}
	fn, ok := legacyDecoders[strings.ToLower(cfg.Go)]
	if !ok {
		return nil, fmt.Errorf("decoder %q: unknown go-ref %q", id, cfg.Go)
	}
	snap.Kind = KindGo
	snap.Go = adapterForGoRef(cfg.Go, withDecodeMetrics(id, fn))
	snap.Hash = configHash(id, cfg, nil)
	return snap, nil
}

func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func withDecodeMetrics(id string, fn DecoderFunc) DecoderFunc {
	messageCounter, err := otel.Meter("iot-agent/decoding").Int64Counter(
		"diwise.decoding."+id+".total",
		metric.WithUnit("1"),
		metric.WithDescription(fmt.Sprintf("Total number of decoded %s payloads", id)),
	)
	if err != nil {
		messageCounter = nil
	}
	errCounter, err := otel.Meter("iot-agent/decoding").Int64Counter(
		"diwise.decoding.errors.total",
		metric.WithUnit("1"),
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		errCounter = nil
	}

	return func(ctx context.Context, e types.Event) (types.SensorPayload, error) {
		p, err := fn(ctx, e)
		if err != nil {
			if errCounter != nil {
				errCounter.Add(ctx, 1)
			}
			return nil, err
		}
		if messageCounter != nil {
			messageCounter.Add(ctx, 1)
		}
		return p, nil
	}
}

func (c *registryImpl) Get(ctx context.Context, sensorType string) (*Snapshot, bool) {
	_ = ctx
	snap, ok := c.snapshots[strings.ToLower(sensorType)]
	return snap, ok
}
