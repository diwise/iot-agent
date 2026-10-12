package decoders

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diwise/iot-agent/internal/application/decoders/mapper"
	"github.com/diwise/iot-agent/internal/application/types"
	"gopkg.in/yaml.v3"
)

// SchemaVersion is the only mappings schema this agent reads.
const SchemaVersion = 1

// DecoderConfig is one decoder-ID entry in the decoders section of
// the unified iot-agent.yaml.
// Exactly one of Go and Script applies. Input defaults to bytes.
type DecoderConfig struct {
	Go     string `yaml:"go,omitempty"`
	Script string `yaml:"script,omitempty"`
	Entry  string `yaml:"entry,omitempty"`
	Input  string `yaml:"input,omitempty"`

	Status mapper.StatusMapping    `yaml:"status,omitempty"`
	Groups map[string]mapper.Group `yaml:"groups,omitempty"`
	Rules  []mapper.Rule           `yaml:"rules,omitempty"`
}

type mappingsFile struct {
	SchemaVersion int                      `yaml:"schemaVersion"`
	Decoders      map[string]DecoderConfig `yaml:"decoders"`
	// DeviceProfiles makes a unified file with 2 top sections loadable.
	// The decoder loader ignores this section; it is parsed by the
	// deviceprofile loader. Unknown top-level fields still fail.
	DeviceProfiles map[string]any `yaml:"deviceprofiles,omitempty"`
}

// LoadMappings reads and structurally validates a mappings file.
// Unknown fields and duplicate keys fail. Script paths resolve
// against the file directory. Group names come from mapping keys.
func LoadMappings(path string) (map[string]DecoderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mappings file: %w", err)
	}

	var file mappingsFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("parse mappings file %s: %w", path, err)
	}
	if file.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("mappings file %s has schemaVersion %d, want %d", path, file.SchemaVersion, SchemaVersion)
	}
	if file.Decoders == nil {
		return nil, fmt.Errorf("mappings file %s has no decoders", path)
	}

	out := make(map[string]DecoderConfig, len(file.Decoders))
	for id, cfg := range file.Decoders {
		if id == "" {
			return nil, fmt.Errorf("mappings file %s has an empty decoder ID", path)
		}
		for name, g := range cfg.Groups {
			g.Name = name
			cfg.Groups[name] = g
		}
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("decoder %q: %w", id, err)
		}
		out[id] = cfg
	}
	return out, nil
}

// Validate checks one entry without file access. Script existence,
// compilation and entrypoints are checked at registry build.
func (c DecoderConfig) Validate() error {
	if c.Go != "" && c.Script != "" {
		return fmt.Errorf("go and script are mutually exclusive")
	}
	if c.Go == "" && c.Script == "" {
		return fmt.Errorf("one of go or script is required")
	}
	if c.Script != "" && c.Entry == "" {
		return fmt.Errorf("script needs an explicit entry")
	}
	if c.Go != "" && c.Entry != "" {
		return fmt.Errorf("entry applies to scripts, not to go")
	}
	switch c.Input {
	case "", "bytes", "object":
	default:
		return fmt.Errorf("input must be bytes or object, got %q", c.Input)
	}
	if c.Go != "" && !knownGoRef(c.Go) {
		return fmt.Errorf("unknown go-ref %q", c.Go)
	}
	if isQalcosonicRef(c.Go) {
		return fmt.Errorf("qalcosonic stays on the legacy path and takes no YAML entry")
	}
	if err := mapper.ValidateRules(c.Rules, c.Groups); err != nil {
		return err
	}
	for _, src := range []mapper.StatusSource{c.Status.Battery, c.Status.Code, c.Status.Messages} {
		if src.Path == "" {
			continue
		}
		if err := types.ValidatePath(src.Path); err != nil {
			return fmt.Errorf("status: %w", err)
		}
		for _, t := range src.Transforms {
			if _, err := mapper.GetTransform(t.Name, t.Params); err != nil {
				return fmt.Errorf("status: %w", err)
			}
		}
	}
	return nil
}

// InputOrDefault resolves the effective input source.
func (c DecoderConfig) InputOrDefault() string {
	if c.Input == "" {
		return "bytes"
	}
	return c.Input
}

// scriptPath resolves a script reference against dir and contains it
// within dir, following symlinks.
func scriptPath(dir, ref string) (string, error) {
	p := ref
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, ref)
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", fmt.Errorf("resolve script %q: %w", ref, err)
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("resolve script root: %w", err)
	}
	if resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
		return "", fmt.Errorf("script %q escapes the script root", ref)
	}
	return resolved, nil
}

// configHash binds an entry revision. It covers normalized config
// and exact script content.
func configHash(id string, cfg DecoderConfig, script []byte) string {
	b, _ := json.Marshal(struct {
		ID     string
		Config DecoderConfig
		Script []byte
	}{id, cfg, script})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
