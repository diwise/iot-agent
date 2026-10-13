package lwm2m

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/matryer/is"
)

// AGENT-005: locks the constructors and serializations consumed by
// iot-core. Every symbol used by iot-core is referenced here so that
// removal or renaming breaks compilation; the fixtures lock the exact
// SenML JSON each constructor produces.
func TestCoreConsumerContract(t *testing.T) {
	is := is.New(t)

	deviceID := "25e185f6-bdba-4c68-b6e8-23ae2bb10254"
	ts := time.Unix(1710151647, 0)

	// Compile-level references to every type iot-core uses.
	var _ Lwm2mObject = Temperature{}
	var _ Lwm2mObject = DigitalInput{}
	var _ Lwm2mObject = Distance{}

	temp := NewTemperature(deviceID, 22.5, ts)
	is.Equal(temp.ObjectID(), "3303")
	is.Equal(temp.ObjectURN(), "urn:oma:lwm2m:ext:3303")
	is.Equal(len(ToPack(temp)), 2)

	input := NewDigitalInput(deviceID, true, ts)
	is.Equal(input.ObjectID(), "3200")
	is.Equal(len(ToPack(input)), 2)

	for _, tc := range []struct {
		name string
		obj  Lwm2mObject
		want string
	}{
		{
			"FillingLevel",
			NewFillingLevel(deviceID, 78.5, ts),
			`[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3435/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3435"},{"n":"1","u":"cm","v":0},{"n":"2","u":"%","v":78.5}]`,
		},
		{
			"PeopleCounter",
			NewPeopleCounter(deviceID, 7, ts),
			`[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3434/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3434"},{"n":"1","v":7},{"n":"2","v":0}]`,
		},
		{
			"Distance",
			NewDistance(deviceID, 3.25, ts),
			`[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3330/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3330"},{"n":"5700","u":"m","v":3.25},{"n":"5701","vs":"metre"}]`,
		},
		{
			"Stopwatch",
			NewStopwatch(deviceID, 90, ts),
			`[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3350/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3350"},{"n":"5544","u":"s","v":90},{"n":"5501","v":0}]`,
		},
		{
			"Timer",
			NewTimer(deviceID, 30, ts),
			`[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3340/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3340"},{"n":"5521","u":"s","v":30},{"n":"5850","vb":false}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			b, err := json.Marshal(tc.obj)
			is.NoErr(err)
			is.Equal(string(b), tc.want)

			pack := ToPack(tc.obj)
			is.True(len(pack) > 1)
			is.Equal(pack[0].BaseName, deviceID+"/"+tc.obj.ObjectID()+"/")
		})
	}
}
