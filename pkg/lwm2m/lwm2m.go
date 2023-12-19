package lwm2m

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/farshidtz/senml/v2"
)

type Lwm2mObject interface {
	ID() string
	ObjectID() string
	ObjectURN() string
	Timestamp() time.Time
	MarshalJSON() ([]byte, error)
}

func ToPack(object Lwm2mObject) senml.Pack {
	o, _ := marshalJSON(object)
	p := senml.Pack{}
	json.Unmarshal(o, &p)

	return p
}

func ToPacks(objects []Lwm2mObject) []senml.Pack {
	packs := []senml.Pack{}

	for _, obj := range objects {
		p := ToPack(obj)
		packs = append(packs, p)
	}

	return packs
}


func Diff(a, b senml.Pack) []senml.Record {
	aa := a.Clone()
	bb := b.Clone()

	aa.Normalize()
	bb.Normalize()

    diff := []senml.Record{}
    recordMap := make(map[string]senml.Record)

    for _, r := range aa {
        recordMap[r.Name] = r
    }

    for _, r2 := range bb {
        r1, exists := recordMap[r2.Name]
        if !exists || !IsEqual(r1, r2) {
            diff = append(diff, r2)
        }
    }

    return diff
}

func IsEqual(a, b senml.Record) bool {
	n := a.Name == b.Name
	u := a.Unit == b.Unit
	vs := a.StringValue == b.StringValue
	vd := a.DataValue == b.DataValue

	vb := isBoolEqual(a.BoolValue, b.BoolValue)

	ts := almostEqual(&a.Time, &b.Time, 0.0001)
	uts := almostEqual(&a.UpdateTime, &b.UpdateTime, 0.0001)

	s := almostEqual(a.Sum, b.Sum, 0.0001)
	val := almostEqual(a.Value, b.Value, 0.0001)

	return n && u && ts && uts && vs && vd && vb && s && val
}

func isBoolEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}

	if a != nil && b != nil {
		return *a == *b
	}

	return false
}

func almostEqual(a, b *float64, epsilon float64) bool {
	if a == nil && b == nil {
		return true
	}

	if a != nil && b != nil {
		return math.Abs(*a-*b) <= epsilon
	}

	return false
}

func Round(val float64) float64 {
	ratio := math.Pow(10, float64(3))
	return math.Round(val*ratio) / ratio
}

func marshalJSON(lwm2mObject Lwm2mObject) ([]byte, error) {
	t := reflect.TypeOf(lwm2mObject)
	v := reflect.ValueOf(lwm2mObject)

	p := senml.Pack{senml.Record{
		BaseName:    fmt.Sprintf("%s/%s/", lwm2mObject.ID(), lwm2mObject.ObjectID()),
		BaseTime:    float64(lwm2mObject.Timestamp().Unix()),
		Name:        "0",
		StringValue: lwm2mObject.ObjectURN(),
	}}

	for i := 0; i < t.NumField(); i++ {
		var tagName, tagUnit = getTags(t.Field(i))

		if tagName == "" || tagName == "-" {
			continue
		}

		value := v.Field(i)

		r := senml.Record{
			Name: tagName,
		}

		if addValue(&r, value) {
			if tagUnit != "" {
				r.Unit = tagUnit
			}
			p = append(p, r)
		}
	}

	return json.Marshal(p)
}

func addValue(r *senml.Record, value reflect.Value) bool {
	kind := value.Kind()

	if kind == reflect.Float32 || kind == reflect.Float64 {
		v := value.Float()
		r.Value = &v
		return true
	}

	if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 {
		v := float64(value.Int())
		r.Value = &v
		return true
	}

	if kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 {
		v := float64(value.Uint())
		r.Value = &v
		return true
	}

	if kind == reflect.String {
		v := value.String()
		r.StringValue = v
		return true
	}

	if kind == reflect.Bool {
		v := value.Bool()
		r.BoolValue = &v
		return true
	}

	if kind == reflect.Ptr || kind == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		return addValue(r, value.Elem())
	}

	return false
}

func getTags(f reflect.StructField) (string, string) {
	tag := f.Tag.Get("lwm2m")
	tags := strings.Split(tag, ",")
	
	if len(tags) > 1 {
		return tags[0], tags[1]
	}
	return tags[0], ""
}
