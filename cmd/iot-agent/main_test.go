package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestParseExternalConfigFile(t *testing.T) {
	is := is.New(t)
	cfg, err := parseExternalConfigFile(context.Background(), io.NopCloser(strings.NewReader(testUnifiedYAML)))
	is.NoErr(err)
	is.Equal(len(cfg), 1)

	elsys := cfg["Elsys_Codec"]
	is.Equal(elsys.ProfileName, "elsys")
	is.Equal(elsys.Tenant, "default")
	is.Equal(elsys.Activate, true)
	is.Equal(elsys.Location, true)
}

func TestParseExternalConfigFileRejectsLegacyBareMap(t *testing.T) {
	is := is.New(t)
	_, err := parseExternalConfigFile(context.Background(), io.NopCloser(strings.NewReader(testLegacyBareMapYAML)))
	is.True(err != nil)
}

func TestParseExternalConfigFileUnified(t *testing.T) {
	is := is.New(t)
	cfg, err := parseExternalConfigFile(context.Background(), io.NopCloser(strings.NewReader(testUnifiedYAML)))
	is.NoErr(err)
	is.Equal(len(cfg), 1)

	elsys := cfg["Elsys_Codec"]
	is.Equal(elsys.ProfileName, "elsys")
	is.Equal(elsys.Tenant, "default")
}

func TestParseExternalConfigFileUnifiedEmptyProfiles(t *testing.T) {
	is := is.New(t)
	cfg, err := parseExternalConfigFile(context.Background(), io.NopCloser(strings.NewReader("schemaVersion: 1\ndeviceprofiles: {}\ndecoders:\n  x:\n    go: senlabt\n    groups: {}\n    rules: []\n")))
	is.NoErr(err)
	is.Equal(len(cfg), 0)
}

const testUnifiedYAML = `
schemaVersion: 1
deviceprofiles:
  Elsys_Codec:
    profile_name: elsys
    tenant: default
    activate: true
    location: true
decoders:
  senlabt:
    go: senlabt
    groups: {}
    rules: []
`

const testLegacyBareMapYAML = `
Elsys_Codec:
  profile_name: elsys
  tenant: default
  activate: true
  location: true
`
