package lwm2m

import (
	"testing"
	"time"

	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestIsEqual(t *testing.T) {
	is := is.New(t)

	ts := time.Now()

	t1 := NewTemperature("test1", 1.0, ts)
	t2 := NewTemperature("test1", 1.0, ts)

	is.True(IsEqual(ToPack(t1)[0], ToPack(t2)[0]))
	is.True(IsEqual(ToPack(t1)[1], ToPack(t2)[1]))

	d1 := NewDigitalInput("test1", true, ts)
	d2 := NewDigitalInput("test1", true, ts)

	is.True(IsEqual(ToPack(d1)[0], ToPack(d2)[0]))
	is.True(IsEqual(ToPack(d1)[1], ToPack(d2)[1]))

	v := 1.0
	vb := true

	r1 := senml.Record{
		Name:        "test1",
		Unit:        "Cel",
		StringValue: "test",
		Time:        float64(ts.Unix()),
		Value:       &v,
		BoolValue:   &vb,
	}
	r2 := senml.Record{
		Name:        "test1",
		Unit:        "Cel",
		StringValue: "test",
		Time:        float64(ts.Unix()),
		Value:       &v,
		BoolValue:   nil,
	}

	is.True(!IsEqual(r1, r2))
}

func TestDiff(t *testing.T) {
	is := is.New(t)

	ts := time.Now()

	t1 := NewTemperature("test1", 1.0, ts)
	t2 := NewTemperature("test1", 2.0, ts)

	diff := Diff(ToPack(t1), ToPack(t2))
	is.Equal(len(diff), 1)
}
