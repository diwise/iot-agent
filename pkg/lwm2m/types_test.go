package lwm2m

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestTemperature(t *testing.T) {
	is := is.New(t)

	deviceID := "25e185f6-bdba-4c68-b6e8-23ae2bb10254"
	ts := time.Unix(1710151647, 0)

	temp := NewTemperature(deviceID, 22.5, ts)

	b, err := json.Marshal(temp)
	is.NoErr(err)

	is.Equal(`[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3303/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":22.5}]`, string(b))
}
