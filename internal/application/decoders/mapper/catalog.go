package mapper

import (
	"fmt"
	"time"

	"github.com/diwise/iot-agent/internal/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

// ValueKind is the Go value kind a resource accepts.
type ValueKind int

const (
	KindFloat ValueKind = iota
	KindInt
	KindBool
	KindString
)

// ResourceDef describes one LWM2M resource in the supported subset.
// Creator resources may create their object. Optional and constant
// resources only complement a created object.
type ResourceDef struct {
	Kind    ValueKind
	Creator bool
}

// ObjectDef describes one supported object type.
type ObjectDef struct {
	URN       string
	Resources map[string]ResourceDef
}

// Catalog holds the actually supported object subset, not the full
// OMA registry. Builder and rule validation share this table.
func Catalog() map[string]ObjectDef {
	floatCreator := ResourceDef{Kind: KindFloat, Creator: true}
	floatOpt := ResourceDef{Kind: KindFloat}
	intCreator := ResourceDef{Kind: KindInt, Creator: true}
	intOpt := ResourceDef{Kind: KindInt}
	boolCreator := ResourceDef{Kind: KindBool, Creator: true}
	stringCreator := ResourceDef{Kind: KindString, Creator: true}
	stringOpt := ResourceDef{Kind: KindString}

	return map[string]ObjectDef{
		"urn:oma:lwm2m:ext:3303": {URN: "urn:oma:lwm2m:ext:3303", Resources: map[string]ResourceDef{
			"5700": floatCreator, "5601": floatOpt, "5602": floatOpt,
			"5603": floatOpt, "5604": floatOpt, "5701": stringOpt, "5750": stringOpt,
		}},
		"urn:oma:lwm2m:ext:3304": {URN: "urn:oma:lwm2m:ext:3304", Resources: map[string]ResourceDef{
			"5700": floatCreator,
		}},
		"urn:oma:lwm2m:ext:3301": {URN: "urn:oma:lwm2m:ext:3301", Resources: map[string]ResourceDef{
			"5700": floatCreator,
		}},
		"urn:oma:lwm2m:ext:3327": {URN: "urn:oma:lwm2m:ext:3327", Resources: map[string]ResourceDef{
			"5700": floatCreator, "5701": stringOpt,
		}},
		"urn:oma:lwm2m:ext:3323": {URN: "urn:oma:lwm2m:ext:3323", Resources: map[string]ResourceDef{
			"5700": floatCreator,
		}},
		"urn:oma:lwm2m:ext:3324": {URN: "urn:oma:lwm2m:ext:3324", Resources: map[string]ResourceDef{
			"5700": floatCreator, "5601": floatOpt, "5602": floatOpt,
		}},
		"urn:oma:lwm2m:ext:3330": {URN: "urn:oma:lwm2m:ext:3330", Resources: map[string]ResourceDef{
			"5700": floatCreator, "5701": stringOpt, "5750": stringOpt,
			"5601": floatOpt, "5602": floatOpt, "5603": floatOpt, "5604": floatOpt,
		}},
		"urn:oma:lwm2m:ext:3200": {URN: "urn:oma:lwm2m:ext:3200", Resources: map[string]ResourceDef{
			"5500": boolCreator, "5501": intOpt,
		}},
		"urn:oma:lwm2m:ext:3302": {URN: "urn:oma:lwm2m:ext:3302", Resources: map[string]ResourceDef{
			"5500": boolCreator, "5501": intOpt,
		}},
		"urn:oma:lwm2m:ext:3428": {URN: "urn:oma:lwm2m:ext:3428", Resources: map[string]ResourceDef{
			"1": floatCreator, "3": floatCreator, "15": floatCreator, "17": floatCreator,
		}},
		"urn:oma:lwm2m:ext:3": {URN: "urn:oma:lwm2m:ext:3", Resources: map[string]ResourceDef{
			"9": intCreator, "7": intCreator, "17": stringCreator,
		}},
		"urn:oma:lwm2m:ext:3435": {URN: "urn:oma:lwm2m:ext:3435", Resources: map[string]ResourceDef{
			"2": floatCreator, "3": intOpt,
		}},
	}
}

// KnownURN reports whether the catalog supports an URN.
func KnownURN(urn string) bool {
	_, ok := Catalog()[urn]
	return ok
}

// AssembleObject builds a typed LWM2M object from coerced values.
// Values must use the kinds from the catalog. Unknown URNs and
// resources fail. Callers validate rules at load; this function
// stays strict for unvalidated input.
func AssembleObject(urn, objectID string, ts time.Time, values map[string]any) (lwm2m.Lwm2mObject, error) {
	def, ok := Catalog()[urn]
	if !ok {
		return nil, fmt.Errorf("unknown URN %q", urn)
	}
	for res := range values {
		if _, ok := def.Resources[res]; !ok {
			return nil, fmt.Errorf("URN %s has no resource %q", urn, res)
		}
	}

	float := func(res string) (float64, bool, error) {
		v, ok := values[res]
		if !ok {
			return 0, false, nil
		}
		f, err := types.AsFloat(v)
		if err != nil {
			return 0, false, fmt.Errorf("resource %q: %w", res, err)
		}
		return f, true, nil
	}
	integer := func(res string) (int, bool, error) {
		v, ok := values[res]
		if !ok {
			return 0, false, nil
		}
		i, err := types.AsInt(v)
		if err != nil {
			return 0, false, fmt.Errorf("resource %q: %w", res, err)
		}
		return i, true, nil
	}
	boolean := func(res string) (bool, bool, error) {
		v, ok := values[res]
		if !ok {
			return false, false, nil
		}
		b, err := types.AsBool(v)
		if err != nil {
			return false, false, fmt.Errorf("resource %q: %w", res, err)
		}
		return b, true, nil
	}

	switch urn {
	case "urn:oma:lwm2m:ext:3303":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		t := lwm2m.NewTemperature(objectID, f, ts)
		setFloatOpt(&t.MinMeasuredValue, values, "5601")
		setFloatOpt(&t.MaxMeasuredValue, values, "5602")
		setFloatOpt(&t.MinRangeValue, values, "5603")
		setFloatOpt(&t.MaxRangeValue, values, "5604")
		setStringOpt(&t.SensorUnits, values, "5701")
		setStringOpt(&t.ApplicationType, values, "5750")
		return t, nil
	case "urn:oma:lwm2m:ext:3304":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		return lwm2m.NewHumidity(objectID, f, ts), nil
	case "urn:oma:lwm2m:ext:3301":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		return lwm2m.NewIlluminance(objectID, f, ts), nil
	case "urn:oma:lwm2m:ext:3327":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		c := lwm2m.NewConductivity(objectID, f, ts)
		setStringOpt(&c.SensorUnits, values, "5701")
		return c, nil
	case "urn:oma:lwm2m:ext:3323":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		return lwm2m.NewPressure(objectID, f, ts), nil
	case "urn:oma:lwm2m:ext:3324":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		l := lwm2m.NewLoudness(objectID, f, ts)
		setFloatOpt(&l.MinMeasuredValue, values, "5601")
		setFloatOpt(&l.MaxMeasuredValue, values, "5602")
		return l, nil
	case "urn:oma:lwm2m:ext:3330":
		f, present, err := float("5700")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5700")
		}
		d := lwm2m.NewDistance(objectID, f, ts)
		setStringOpt(&d.SensorUnits, values, "5701")
		setStringOpt(&d.ApplicationType, values, "5750")
		setFloatOpt(&d.MinMeasuredValue, values, "5601")
		setFloatOpt(&d.MaxMeasuredValue, values, "5602")
		setFloatOpt(&d.MinRangeValue, values, "5603")
		setFloatOpt(&d.MaxRangeValue, values, "5604")
		return d, nil
	case "urn:oma:lwm2m:ext:3200":
		b, present, err := boolean("5500")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5500")
		}
		d := lwm2m.NewDigitalInput(objectID, b, ts)
		if i, ok, err := integer("5501"); err != nil {
			return nil, err
		} else if ok {
			d.DigitalInputCounter = &i
		}
		return d, nil
	case "urn:oma:lwm2m:ext:3302":
		b, present, err := boolean("5500")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "5500")
		}
		p := lwm2m.NewPresence(objectID, b, ts)
		if i, ok, err := integer("5501"); err != nil {
			return nil, err
		} else if ok {
			p.DigitalInputCounter = &i
		}
		return p, nil
	case "urn:oma:lwm2m:ext:3428":
		var co2, pm10, pm25, no2 *float64
		for res, dst := range map[string]**float64{"17": &co2, "1": &pm10, "3": &pm25, "15": &no2} {
			if f, ok, err := float(res); err != nil {
				return nil, err
			} else if ok {
				*dst = &f
			}
		}
		return lwm2m.NewAirQuality(objectID, co2, pm10, pm25, no2, ts), nil
	case "urn:oma:lwm2m:ext:3":
		d := lwm2m.NewDevice(objectID, ts)
		if i, ok, err := integer("9"); err != nil {
			return nil, err
		} else if ok {
			d.BatteryLevel = &i
		}
		if i, ok, err := integer("7"); err != nil {
			return nil, err
		} else if ok {
			d.PowerSourceVoltage = &i
		}
		setStringOpt(&d.DeviceType, values, "17")
		return d, nil
	case "urn:oma:lwm2m:ext:3435":
		f, present, err := float("2")
		if err != nil || !present {
			return nil, errOrMissing(err, urn, "2")
		}
		fl := lwm2m.NewFillingLevel(objectID, f, ts)
		if i, ok, err := integer("3"); err != nil {
			return nil, err
		} else if ok {
			cm := int64(i)
			fl.ActualFillingLevel = &cm
		}
		return fl, nil
	default:
		return nil, fmt.Errorf("unknown URN %q", urn)
	}
}

func errOrMissing(err error, urn, res string) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("URN %s needs resource %q", urn, res)
}

func setFloatOpt(dst **float64, values map[string]any, res string) {
	if v, ok := values[res]; ok {
		if f, err := types.AsFloat(v); err == nil {
			*dst = &f
		}
	}
}

func setStringOpt(dst **string, values map[string]any, res string) {
	if v, ok := values[res]; ok {
		if s, ok := v.(string); ok {
			*dst = &s
		}
	}
}
