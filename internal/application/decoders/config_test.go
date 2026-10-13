package decoders

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func writeMappings(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mappings.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMappingsGoEntry(t *testing.T) {
	is := is.New(t)

	path := writeMappings(t, `
schemaVersion: 1
decoders:
  milesight:
    go: milesight
    input: bytes
    groups:
      temp:
        urn: urn:oma:lwm2m:ext:3303
    rules:
      - id: t
        from: temperature
        group: temp
        resource: "5700"
`)
	cfgs, err := LoadMappings(path)
	is.NoErr(err)
	cfg := cfgs["milesight"]
	is.Equal(cfg.Go, "milesight")
	is.Equal(cfg.InputOrDefault(), "bytes")
	is.Equal(cfg.Groups["temp"].Name, "temp")
}

func TestLoadMappingsDefaultsInput(t *testing.T) {
	is := is.New(t)

	path := writeMappings(t, `
schemaVersion: 1
decoders:
  x:
    go: senlabt
    groups: {}
    rules: []
`)
	cfgs, err := LoadMappings(path)
	is.NoErr(err)
	is.Equal(cfgs["x"].InputOrDefault(), "bytes")
}

func TestLoadMappingsRejects(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		name    string
		content string
	}{
		{"unknown field", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    bogus: 1\n    groups: {}\n    rules: []\n"},
		{"duplicate key", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    go: milesight\n    groups: {}\n    rules: []\n"},
		{"bad version", "schemaVersion: 2\ndecoders:\n  x:\n    go: senlabt\n    groups: {}\n    rules: []\n"},
		{"missing version", "decoders:\n  x:\n    go: senlabt\n    groups: {}\n    rules: []\n"},
		{"go and script", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    script: a.js\n    entry: ttn-decodeUplink\n    groups: {}\n    rules: []\n"},
		{"neither go nor script", "schemaVersion: 1\ndecoders:\n  x:\n    groups: {}\n    rules: []\n"},
		{"script without entry", "schemaVersion: 1\ndecoders:\n  x:\n    script: a.js\n    groups: {}\n    rules: []\n"},
		{"go with entry", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    entry: ttn-decodeUplink\n    groups: {}\n    rules: []\n"},
		{"bad input", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    input: smoke\n    groups: {}\n    rules: []\n"},
		{"unknown go-ref", "schemaVersion: 1\ndecoders:\n  x:\n    go: nope\n    groups: {}\n    rules: []\n"},
		{"qalcosonic go-ref", "schemaVersion: 1\ndecoders:\n  x:\n    go: qalcosonic\n    groups: {}\n    rules: []\n"},
		{"unknown resource", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    groups:\n      temp:\n        urn: urn:oma:lwm2m:ext:3303\n    rules:\n      - id: t\n        from: temperature\n        group: temp\n        resource: \"9999\"\n"},
		{"bad status path", "schemaVersion: 1\ndecoders:\n  x:\n    go: senlabt\n    status:\n      battery:\n        path: \"a[\"\n    groups: {}\n    rules: []\n"},
	}

	for _, tc := range cases {
		_, err := LoadMappings(writeMappings(t, tc.content))
		is.True(err != nil)
	}
}

func TestLoadMappingsMissingFile(t *testing.T) {
	is := is.New(t)

	_, err := LoadMappings(filepath.Join(t.TempDir(), "nope.yaml"))
	is.True(err != nil)
}

func TestLoadMappingsIgnoresDeviceprofilesSection(t *testing.T) {
	is := is.New(t)

	path := writeMappings(t, `
schemaVersion: 1
deviceprofiles:
  Elsys_Codec:
    profile_name: elsys
    tenant: default
    activate: true
    location: true
decoders:
  milesight:
    go: milesight
    input: bytes
    groups:
      temp:
        urn: urn:oma:lwm2m:ext:3303
    rules:
      - id: t
        from: temperature
        group: temp
        resource: "5700"
`)
	cfgs, err := LoadMappings(path)
	is.NoErr(err)
	is.Equal(cfgs["milesight"].Go, "milesight")
}
