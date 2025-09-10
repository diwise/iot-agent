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
	cfg, err := parseExternalConfigFile(context.Background(), io.NopCloser(strings.NewReader(testDeviceProfileYAML)))
	is.NoErr(err)
	is.Equal(len(cfg.Profiles), 2)
}

const testDeviceProfileYAML = `
profiles:
    - sensor_type: Elsys_Codec
      profile_name: elsys
      tenant: default
      activate: true
      location: true
      tags: true
    - sensor_type: Axioma_Universal_Codec
      profile_name: qalcosonic
      tenant: default
      activate: true
      location: false
      tags: false
`
