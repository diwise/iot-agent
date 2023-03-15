package payload

import (
	"math"
	"reflect"
	"strings"
	"time"
)

var PayloadError = 100

type PayloadDecoratorFunc func(p *PayloadImpl)

type Payload interface {
	DevEui() string
	Timestamp() time.Time
	Status() StatusImpl
	Get(name string) (any, bool)
}

const (
	BatteryLevelProperty string = "batteryLevel"
	CO2Property          string = "co2"
	DistanceProperty     string = "distance"
	HumidityProperty     string = "humidity"
	InputCounterProperty string = "digitalInputCounter"
	InputStateProperty   string = "digitalInputState"
	LightProperty        string = "light"
	MotionProperty       string = "motion"
	OccupancyProperty    string = "occupancy"
	PresenceProperty     string = "presence"
	PressureProperty     string = "pressure"
	ResistanceProperty   string = "resistance"
	StatusProperty       string = "status"
	TemperatureProperty  string = "temperature"
	TimestampProperty    string = "timestamp"
	TypeProperty         string = "type"
	VolumeProperty       string = "volume"

	DoorAlarmProperty  string = "doorAlarm"
	DoorReportProperty string = "doorReport"
	SnowHeightProperty string = "snowHeight"

	FrameVersionProperty       string = "frameVersion"
	ProtocolVersionProperty    string = "protocolVersion"
	TransmissionReasonProperty string = "transmissionReason"
)

type PayloadImpl struct {
	devEui       string
	measurements map[string]any
	timestamp    time.Time
}

type StatusImpl struct {
	Code     int
	Messages []string
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

func (p *PayloadImpl) DevEui() string {
	return p.devEui
}

func (p *PayloadImpl) Timestamp() time.Time {
	return p.timestamp
}

func (p *PayloadImpl) Status() StatusImpl {
	if s, ok := p.Get(StatusProperty); ok {
		if si, ok := s.(StatusImpl); ok {
			return si
		}
	}
	return StatusImpl{}
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

func GetSlice[T any](p Payload, key string) ([]T, bool) {
	data := make([]T, 0)
	if values, ok := p.Get(key); ok {
		if _values, ok := values.([]interface{}); ok {
			for _, v := range _values {
				if d, ok := v.(T); ok {
					data = append(data, d)
				} else {
					return nil, false
				}
			}
		} else {
			if _values, ok := values.(T); ok {
				data = append(data, _values)
			} else {
				return nil, false
			}
		}
	} else {
		return nil, false
	}

	if len(data) > 0 {
		return data, true
	}

	return nil, false
}

func BatteryVoltage(b int) PayloadDecoratorFunc {
	return BatteryLevel(b)
}

func BatteryLevel(b int) PayloadDecoratorFunc {
	return S(BatteryLevelProperty, struct {
		BatteryLevel int
	}{
		b,
	})
}

func Temperature(t float64) PayloadDecoratorFunc {
	roundFloat := func(val float64) float64 {
		ratio := math.Pow(10, float64(5))
		return math.Round(val*ratio) / ratio
	}

	return S(TemperatureProperty, struct {
		Temperature float64
	}{
		roundFloat(t),
	})
}

func CO2(co2 int) PayloadDecoratorFunc {
	return S(CO2Property, struct {
		CO2 int
	}{
		co2,
	})
}

func Distance(d float64) PayloadDecoratorFunc {
	return S(DistanceProperty, struct {
		Distance float64
	}{
		d,
	})
}

func Humidity(h int) PayloadDecoratorFunc {
	return S(HumidityProperty, struct {
		Humidity int
	}{
		h,
	})
}

func Light(l int) PayloadDecoratorFunc {
	return S(LightProperty, struct {
		Light int
	}{
		l,
	})
}

func Motion(m int) PayloadDecoratorFunc {
	return S(MotionProperty, struct {
		Motion int
	}{
		m,
	})
}

func Status(c uint8, msg []string) PayloadDecoratorFunc {
	return S(StatusProperty, StatusImpl{
		Code:     int(c),
		Messages: msg,
	})
}

func Volume(v, c float64, t time.Time) PayloadDecoratorFunc {
	return M(VolumeProperty, struct {
		Volume    float64
		Cumulated float64
		Time      time.Time
	}{
		v, c, t,
	})
}

func FrameVersion(fv uint8) PayloadDecoratorFunc {
	return S(FrameVersionProperty, struct {
		FrameVersion int `json:"frameVersion"`
	}{
		FrameVersion: int(fv),
	})
}

func Presence(p bool) PayloadDecoratorFunc {
	return S(PresenceProperty, struct {
		Presence bool
	}{
		p,
	})
}

func SnowHeight(sh int) PayloadDecoratorFunc {
	return S(SnowHeightProperty, struct {
		SnowHeight int
	}{
		sh,
	})
}

func DigitalInputCounter(count int64) PayloadDecoratorFunc {
	return S(InputCounterProperty, struct {
		DigitalInputCounter int64
	}{
		count,
	})
}

func DigitalInputState(on bool) PayloadDecoratorFunc {
	return S(InputStateProperty, struct {
		DigitalInputState bool
	}{
		on,
	})
}

func DoorReport(p bool) PayloadDecoratorFunc {
	// TODO: Return DigitalInputState ?
	return S(DoorReportProperty, struct {
		DoorReport bool
	}{
		p,
	})
}

func DoorAlarm(p bool) PayloadDecoratorFunc {
	return S(DoorAlarmProperty, struct {
		DoorAlarm bool
	}{
		p,
	})
}

func TransmissionReason(tr int8) PayloadDecoratorFunc {
	return S(TransmissionReasonProperty, struct {
		TransmissionReason int8
	}{
		tr,
	})
}

func ProtocolVersion(v int8) PayloadDecoratorFunc {
	return S(ProtocolVersionProperty, struct {
		ProtocolVersion int8
	}{
		v,
	})
}

func Resistance(r int32) PayloadDecoratorFunc {
	return M(ResistanceProperty, struct {
		Resistance int32
	}{
		r,
	})
}

func Occupancy(p int) PayloadDecoratorFunc {
	return S(OccupancyProperty, struct {
		Occupancy int
	}{
		p,
	})
}

// Pressure accepts a pressure value in Pascal
func Pressure(p int16) PayloadDecoratorFunc {
	return M(PressureProperty, struct {
		Pressure int16
	}{
		p,
	})
}

func Type(t string) PayloadDecoratorFunc {
	return S(TypeProperty, struct {
		Type string
	}{
		t,
	})
}

func Timestamp(t time.Time) PayloadDecoratorFunc {
	return S(TimestampProperty, struct {
		Timestamp time.Time
	}{
		t,
	})
}
