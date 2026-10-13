package decoders

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestNewRegistryLegacySnapshots(t *testing.T) {
	is := is.New(t)

	r := NewRegistry()
	ctx := t.Context()

	snap, ok := r.Get(ctx, "elsys")
	is.True(ok)
	is.Equal(snap.Kind, KindLegacy)
	is.True(snap.Legacy.Decoder != nil)
	is.True(snap.Legacy.Converter != nil)

	snap, ok = r.Get(ctx, "qalcosonic/w1t")
	is.True(ok)
	is.Equal(snap.Kind, KindLegacy)

	// Case-insensitive lookup.
	_, ok = r.Get(ctx, "ELYSYS")
	is.True(!ok)
	_, ok = r.Get(ctx, "ELSYS")
	is.True(ok)

	// Unknown IDs stay unknown.
	_, ok = r.Get(ctx, "nope")
	is.True(!ok)
}

func writeTempMappings(t *testing.T, files map[string]string, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "mappings.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const testScript = `function decodeUplink(input) { return {data: {temperature: 20}}; }`

// fullCoverageYAML returns minimal valid entries for every required
// ID so strict BuildRegistry passes. Callers can append extra entries.
func fullCoverageYAML(extra string) string {
	base := "schemaVersion: 1\ndecoders:\n"
	for _, id := range RequiredIDs() {
		goRef := id
		input := ""
		if id == "enviot" || id == "niab-fls" {
			input = "    input: object\n"
		}
		base += "  " + id + ":\n    go: " + goRef + "\n" + input + "    groups: {}\n    rules: []\n"
	}
	if extra != "" {
		base += extra
	}
	return base
}

func TestBuildRegistryJS(t *testing.T) {
	is := is.New(t)

	extra := `  x/model:
    script: model.js
    entry: ttn-decodeUplink
    groups:
      temp:
        urn: urn:oma:lwm2m:ext:3303
    rules:
      - id: t
        from: temperature
        group: temp
        resource: "5700"
`
	path := writeTempMappings(t,
		map[string]string{"model.js": testScript},
		fullCoverageYAML(extra))
	r, err := BuildRegistry(t.Context(), path)
	is.NoErr(err)

	snap, ok := r.Get(t.Context(), "x/model")
	is.True(ok)
	is.Equal(snap.Kind, KindJS)
	is.Equal(snap.Input, InputBytes)
	is.True(snap.Hash != "")
	is.Equal(len(snap.Rules), 1)

	// Required IDs use the generic path, Qalcosonic stays legacy.
	snap, ok = r.Get(t.Context(), "senlabt")
	is.True(ok)
	is.Equal(snap.Kind, KindGo)

	snap, ok = r.Get(t.Context(), "qalcosonic")
	is.True(ok)
	is.Equal(snap.Kind, KindLegacy)

	_, ok = r.Get(t.Context(), "nope")
	is.True(!ok)
}

func TestBuildRegistryGo(t *testing.T) {
	is := is.New(t)

	path := writeTempMappings(t, nil, fullCoverageYAML(""))
	r, err := BuildRegistry(t.Context(), path)
	is.NoErr(err)

	snap, ok := r.Get(t.Context(), "senlabt")
	is.True(ok)
	is.Equal(snap.Kind, KindGo)
	is.True(snap.Go != nil)
	is.True(snap.Hash != "")
}

func TestBuildRegistryStrictMissing(t *testing.T) {
	is := is.New(t)

	path := writeTempMappings(t, nil, `schemaVersion: 1
decoders:
  senlabt:
    go: senlabt
    groups: {}
    rules: []
`)
	_, err := BuildRegistry(t.Context(), path)
	is.True(err != nil)
}

func TestBuildRegistryRejects(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Qalcosonic takes no YAML entry.
	path := writeTempMappings(t, nil, "schemaVersion: 1\ndecoders:\n  qalcosonic:\n    go: senlabt\n    groups: {}\n    rules: []\n")
	_, err := BuildRegistry(ctx, path)
	is.True(err != nil)

	// Missing script file fails.
	path = writeTempMappings(t, nil, "schemaVersion: 1\ndecoders:\n  x:\n    script: nope.js\n    entry: ttn-decodeUplink\n    groups: {}\n    rules: []\n")
	_, err = BuildRegistry(ctx, path)
	is.True(err != nil)

	// Uncompilable script fails.
	path = writeTempMappings(t, map[string]string{"bad.js": "var x = ;"}, "schemaVersion: 1\ndecoders:\n  x:\n    script: bad.js\n    entry: ttn-decodeUplink\n    groups: {}\n    rules: []\n")
	_, err = BuildRegistry(ctx, path)
	is.True(err != nil)

	// Script escaping the script root fails.
	path = writeTempMappings(t, nil, "schemaVersion: 1\ndecoders:\n  x:\n    script: ../evil.js\n    entry: ttn-decodeUplink\n    groups: {}\n    rules: []\n")
	_, err = BuildRegistry(ctx, path)
	is.True(err != nil)

	// Object input needs an object-capable go-ref.
	path = writeTempMappings(t, nil, "schemaVersion: 1\ndecoders:\n  milesight:\n    go: milesight\n    input: object\n    groups: {}\n    rules: []\n")
	_, err = BuildRegistry(ctx, path)
	is.True(err != nil)

	// Scripts support only bytes input.
	path = writeTempMappings(t, map[string]string{"m.js": testScript}, "schemaVersion: 1\ndecoders:\n  x:\n    script: m.js\n    entry: ttn-decodeUplink\n    input: object\n    groups: {}\n    rules: []\n")
	_, err = BuildRegistry(ctx, path)
	is.True(err != nil)

	// Case-colliding keys fail.
	path = writeTempMappings(t, nil, "schemaVersion: 1\ndecoders:\n  Abc:\n    go: senlabt\n    groups: {}\n    rules: []\n  aBC:\n    go: senlabt\n    groups: {}\n    rules: []\n")
	_, err = BuildRegistry(ctx, path)
	is.True(err != nil)
}

func TestBuildRegistryObjectInput(t *testing.T) {
	is := is.New(t)

	path := writeTempMappings(t, nil, fullCoverageYAML(""))
	r, err := BuildRegistry(t.Context(), path)
	is.NoErr(err)

	snap, ok := r.Get(t.Context(), "enviot")
	is.True(ok)
	is.Equal(snap.Kind, KindGo)
	is.Equal(snap.Input, InputObject)
}
