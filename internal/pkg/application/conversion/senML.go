package conversion

import (
	"time"

	"github.com/farshidtz/senml/v2"
)

type SenMLDecoratorFunc func(p *senML)

type senML struct {
	Pack senml.Pack
}

func NewSenMLPack(deviceID, baseName string, baseTime time.Time, decorators ...SenMLDecoratorFunc) senml.Pack {
	s := &senML{}

	s.Pack = append(s.Pack, senml.Record{
		BaseName:    baseName,
		BaseTime:    float64(baseTime.Unix()),
		Name:        "0",
		StringValue: deviceID,
	})

	for _, d := range decorators {
		d(s)
	}

	return s.Pack
}

func Value(n string, v float64) SenMLDecoratorFunc {
	return func(p *senML) {
		r := senml.Record{
			Name:  n,
			Value: &v,
		}
		p.Pack = append(p.Pack, r)
	}
}

func StringValue(n string, vs string) SenMLDecoratorFunc {
	return func(p *senML) {
		r := senml.Record{
			Name:        n,
			StringValue: vs,
		}
		p.Pack = append(p.Pack, r)
	}
}

func BoolValue(n string, vb bool) SenMLDecoratorFunc {
	return func(p *senML) {
		r := senml.Record{
			Name:      n,
			BoolValue: &vb,
		}
		p.Pack = append(p.Pack, r)
	}
}

func Time(n string, t time.Time) SenMLDecoratorFunc {
	return func(p *senML) {
		r := senml.Record{
			Name:        n,
			StringValue: t.Format(time.RFC3339Nano),
			Time:        float64(t.Unix()),
		}
		p.Pack = append(p.Pack, r)
	}
}

func DeltaVolume(v, s float64, t time.Time) SenMLDecoratorFunc {
	return func(p *senML) {
		r := senml.Record{
			Name:  "DeltaVolume",
			Value: &v,
			Time:  float64(t.Unix()),
			Sum:   &s,
		}
		p.Pack = append(p.Pack, r)
	}
}