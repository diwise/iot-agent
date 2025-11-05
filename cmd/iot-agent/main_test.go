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
	is.Equal(len(cfg), 3)

	elsys := cfg["Elsys_Codec"]
	is.Equal(elsys.ProfileName, "elsys")
	is.Equal(elsys.Tenant, "default")
	is.Equal(elsys.Activate, true)
	is.Equal(elsys.Location, true)
	is.Equal(elsys.Tags.Enabled, true)
	is.Equal(elsys.Tags.Metadata, true)
	is.Equal(len(elsys.Tags.Mappings), 2)
	is.Equal(elsys.Tags.Mappings["location"], "plats")
	is.Equal(elsys.Tags.Mappings["mount"], "position")
}

const testDeviceProfileYAML = `
Elsys_Codec:
  profile_name: elsys
  tenant: default
  activate: true
  location: true
  tags: 
    enabled: true
    metadata: true
    mappings:
      location: plats
      mount: position
Axioma_Universal_Codec:
  profile_name: qalcosonic
  tenant: default
  activate: true
  location: false
Milesight EM400 UDL:
  profile_name: milesight
  tenant: default
  activate: true
  location: true
`
