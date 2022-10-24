package payload

import (
	"reflect"
	"strings"
	"time"
)

var PayloadError = 100

type PayloadDecoratorFunc func(p *PayloadImpl)

func S(name string, value any) PayloadDecoratorFunc {
	name = strings.ToLower(name)
	return func(p *PayloadImpl) {
		p.measurements[name] = value
	}
}
func M(name string, value any) PayloadDecoratorFunc {
	name = strings.ToLower(name)
	return func(p *PayloadImpl) {
		if _, ok := p.measurements[name]; ok {
			p.measurements[name] = append(p.measurements[name].([]any), value)
		} else {
			p.measurements[name] = []any{value}
		}
	}
}

func New(devEui string, timestamp time.Time, decorators ...PayloadDecoratorFunc) (Payload, error) {
	p := &PayloadImpl{
		devEui:       devEui,
		timestamp:    timestamp,
		measurements: make(map[string]any),
	}
	for _, decorator := range decorators {
		decorator(p)
	}
	return p, nil
}

type Payload interface {
	DevEui() string
	Timestamp() time.Time
	Status() StatusImpl
	Measurements() []any
	ValueOf(name string) (any, bool)
	Get(name string) (any, bool)
}

type PayloadImpl struct {
	devEui       string
	measurements map[string]any
	status       StatusImpl
	timestamp    time.Time
}

type StatusImpl struct {
	Code     int
	Messages []string
}

func (p *PayloadImpl) DevEui() string {
	return p.devEui
}
func (p *PayloadImpl) Timestamp() time.Time {
	return p.timestamp
}
func (p *PayloadImpl) Status() StatusImpl {
	return p.status
}
func (p *PayloadImpl) Measurements() []any {
	var m []any
	for _, v := range p.measurements {
		m = append(m, v)
	}
	return m
}
func (p *PayloadImpl) ValueOf(name string) (any, bool) {
	name = strings.ToLower(name)

	reflectValue := func(m any) (any, bool) {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				if strings.EqualFold(t.Field(i).Name, name) {
					v := reflect.ValueOf(m)
					return v.Field(i).Interface(), true
				}
			}
		}

		return nil, false
	}

	if m, ok := p.measurements[name]; ok {
		return reflectValue(m)
	}

	for _, m := range p.measurements {
		if v, ok := reflectValue(m); ok {
			return v, ok
		}
	}

	return nil, false
}
func (p *PayloadImpl) Get(name string) (any, bool) {
	name = strings.ToLower(name)
	if m, ok := p.measurements[name]; ok {
		return m, ok
	}
	return nil, false
}
func Get[T any](p Payload, name string) (T, bool) {
	var result T
	if m, ok := p.Get(name); ok {
		if t := reflect.TypeOf(m); t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				if strings.EqualFold(t.Field(i).Name, name) {
					v := reflect.ValueOf(m)
					if val, ok := v.Field(i).Interface().(T); ok {
						return val, true
					} else {
						return result, false
					}
				}
			}
		}
	}

	return result, false
}
func BatteryVoltage(b int) PayloadDecoratorFunc {
	return BatteryLevel(b)
}
func BatteryLevel(b int) PayloadDecoratorFunc {
	return S("batteryLevel", struct {
		BatteryLevel int
	}{
		b,
	})
}
func Temperature(t float32) PayloadDecoratorFunc {
	return S("temperature", struct {
		Temperature float32
	}{
		t,
	})
}
func CO2(co2 int) PayloadDecoratorFunc {
	return S("co2", struct {
		CO2 int
	}{
		co2,
	})
}
func Humidity(h int) PayloadDecoratorFunc {
	return S("humidity", struct {
		Humidity int
	}{
		h,
	})
}
func Light(l int) PayloadDecoratorFunc {
	return S("light", struct {
		Light int
	}{
		l,
	})
}
func Motion(m int) PayloadDecoratorFunc {
	return S("motion", struct {
		Motion int
	}{
		m,
	})
}
func CurrentTime(t time.Time) PayloadDecoratorFunc {
	return S("currentTime", struct {
		CurrentTime time.Time
	}{
		t,
	})
}
func Status(c uint8, msg []string) PayloadDecoratorFunc {
	return S("status", struct {
		StatusCode     uint8
		StatusMessages []string
	}{
		c,
		msg,
	})
}
func CurrentVolume(v float64) PayloadDecoratorFunc {
	return S("currentVolume", struct {
		CurrentVolume float64
	}{
		v,
	})
}
func LogDateTime(d time.Time) PayloadDecoratorFunc {
	return S("logDateTime", struct {
		LogDateTime time.Time
	}{
		d,
	})
}
func LastLogValue(v float64) PayloadDecoratorFunc {
	return S("lastLogValue", struct {
		LastLogValue float64
	}{
		v,
	})
}
func DeltaVolume(v, c float64, t time.Time) PayloadDecoratorFunc {
	return M("deltaVolume", struct {
		Delta        float64
		Cumulated    float64
		LogValueDate time.Time
	}{
		v, c, t,
	})
}
func FrameVersion(fv uint8) PayloadDecoratorFunc {
	return S("frameVersion", struct {
		FrameVersion int `json:"frameVersion"`
	}{
		FrameVersion: int(fv),
	})
}
func Presence(p bool) PayloadDecoratorFunc {
	return S("presence", struct {
		Presence bool
	}{
		p,
	})
}
func SnowHeight(sh int) PayloadDecoratorFunc {
	return S("presence", struct {
		SnowHeight int
	}{
		sh,
	})
}
func DoorReport(p bool) PayloadDecoratorFunc {
	return S("doorReport", struct {
		DoorReport bool
	}{
		p,
	})
}
func DoorAlarm(p bool) PayloadDecoratorFunc {
	return S("doorAlarm", struct {
		DoorAlarm bool
	}{
		p,
	})
}
func TransmissionReason(tr int8) PayloadDecoratorFunc {
	return S("transmissionReason", struct {
		TransmissionReason int8
	}{
		tr,
	})
}
func ProtocolVersion(v int8) PayloadDecoratorFunc {
	return S("protocolVersion", struct {
		ProtocolVersion int8
	}{
		v,
	})
}
func Resistance(r []int32) PayloadDecoratorFunc {
	return S("resistance", struct {
		Resistance []int32
	}{
		r,
	})
}
func SoilMoisture(sm []int16) PayloadDecoratorFunc {
	return S("soilMoisture", struct {
		SoilMoisture []int16
	}{
		sm,
	})
}
