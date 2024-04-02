package lwm2m

import (
	"fmt"
	"time"
)

const prefix = "urn:oma:lwm2m:ext"

type DeviceInfo struct {
	ID_        string    `lwm2m:"-"`
	Timestamp_ time.Time `lwm2m:"-"`
}

func NewTemperature(deviceID string, sensorValue float64, ts time.Time) Temperature {
	return Temperature{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: Round(sensorValue),
	}
}

type Temperature struct {
	DeviceInfo
	SensorValue      float64  `lwm2m:"5700,Cel"`
	MinMeasuredValue *float64 `lwm2m:"5601,Cel"`
	MaxMeasuredValue *float64 `lwm2m:"5602,Cel"`
	MinRangeValue    *float64 `lwm2m:"5603,Cel"`
	MaxRangeValue    *float64 `lwm2m:"5604,Cel"`
	SensorUnits      *string  `lwm2m:"5701"`
	ApplicationType  *string  `lwm2m:"5750"`
}

func (t Temperature) ID() string {
	return t.ID_
}
func (t Temperature) Timestamp() time.Time {
	return t.Timestamp_
}
func (t Temperature) ObjectID() string {
	return "3303"
}
func (t Temperature) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, t.ObjectID())
}
func (t Temperature) MarshalJSON() ([]byte, error) {
	return marshalJSON(t)
}

func NewHumidity(deviceID string, sensorValue float64, ts time.Time) Humidity {
	return Humidity{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Humidity struct {
	DeviceInfo
	SensorValue float64 `lwm2m:"5700,%RH"`
}

func (h Humidity) ID() string {
	return h.ID_
}
func (h Humidity) Timestamp() time.Time {
	return h.Timestamp_
}
func (h Humidity) ObjectID() string {
	return "3304"
}
func (h Humidity) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, h.ObjectID())
}
func (h Humidity) MarshalJSON() ([]byte, error) {
	return marshalJSON(h)
}

func NewIlluminance(deviceID string, sensorValue float64, ts time.Time) Illuminance {
	return Illuminance{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Illuminance struct {
	DeviceInfo
	SensorValue float64 `lwm2m:"5700,lux"`
}

func (i Illuminance) ID() string {
	return i.ID_
}
func (i Illuminance) Timestamp() time.Time {
	return i.Timestamp_
}
func (i Illuminance) ObjectID() string {
	return "3301"
}
func (i Illuminance) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, i.ObjectID())
}
func (i Illuminance) MarshalJSON() ([]byte, error) {
	return marshalJSON(i)
}

func NewAirQuality(deviceID string, co2 float64, ts time.Time) AirQuality {
	return AirQuality{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		CO2: &co2,
	}
}

type AirQuality struct {
	DeviceInfo
	CO2 *float64 `lwm2m:"17,ppm"`
}

func (aq AirQuality) ID() string {
	return aq.ID_
}
func (aq AirQuality) Timestamp() time.Time {
	return aq.Timestamp_
}
func (aq AirQuality) ObjectID() string {
	return "3428"
}
func (aq AirQuality) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, aq.ObjectID())
}
func (aq AirQuality) MarshalJSON() ([]byte, error) {
	return marshalJSON(aq)
}

func NewWaterMeter(deviceID string, cumulatedWaterVolume float64, ts time.Time) WaterMeter {
	return WaterMeter{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		CumulatedWaterVolume: cumulatedWaterVolume,
	}
}

type WaterMeter struct {
	DeviceInfo
	CumulatedWaterVolume float64  `lwm2m:"1,m3"`
	TypeOfMeter          *string  `lwm2m:"3"`
	CumulatedPulseValue  *int     `lwm2m:"4"`
	PulseRatio           *int     `lwm2m:"5"`
	MinimumFlowRate      *float64 `lwm2m:"7,m3/s"`
	MaximumFlowRate      *float64 `lwm2m:"8,m3/s"`
	LeakSuspected        *bool    `lwm2m:"9"`
	LeakDetected         *bool    `lwm2m:"10"`
	BackFlowDetected     *bool    `lwm2m:"11"`
	BlockedMeter         *bool    `lwm2m:"12"`
	FraudDetected        *bool    `lwm2m:"13"`
}

func (w WaterMeter) ID() string {
	return w.ID_
}
func (w WaterMeter) Timestamp() time.Time {
	return w.Timestamp_
}
func (w WaterMeter) ObjectID() string {
	return "3424"
}
func (w WaterMeter) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, w.ObjectID())
}
func (w WaterMeter) MarshalJSON() ([]byte, error) {
	return marshalJSON(w)
}

func NewBattery(deviceID string, batteryLevel int, ts time.Time) Battery {
	return Battery{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		BatteryLevel: batteryLevel,
	}
}

type Battery struct {
	DeviceInfo
	BatteryLevel    int      `lwm2m:"1,%"`
	BatteryCapacity *float64 `lwm2m:"2,Ah"`
	BatteryVoltage  *float64 `lwm2m:"3,V"`
}

func (b Battery) ID() string {
	return b.ID_
}
func (b Battery) Timestamp() time.Time {
	return b.Timestamp_
}
func (b Battery) ObjectID() string {
	return "3411"
}
func (b Battery) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, b.ObjectID())
}
func (b Battery) MarshalJSON() ([]byte, error) {
	return marshalJSON(b)
}

func NewDigitalInput(deviceID string, digitalInputState bool, ts time.Time) DigitalInput {
	return DigitalInput{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		DigitalInputState: digitalInputState,
	}
}

type DigitalInput struct {
	DeviceInfo
	DigitalInputState   bool `lwm2m:"5500"`
	DigitalInputCounter *int `lwm2m:"5501"`
}

func (d DigitalInput) ID() string {
	return d.ID_
}
func (d DigitalInput) Timestamp() time.Time {
	return d.Timestamp_
}
func (d DigitalInput) ObjectID() string {
	return "3200"
}
func (d DigitalInput) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, d.ObjectID())
}
func (d DigitalInput) MarshalJSON() ([]byte, error) {
	return marshalJSON(d)
}

func NewPeopleCounter(deviceID string, actualNumberOfPersons int, ts time.Time) PeopleCounter {
	return PeopleCounter{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		ActualNumberOfPersons: actualNumberOfPersons,
	}
}

type PeopleCounter struct {
	DeviceInfo
	ActualNumberOfPersons int `lwm2m:"1"`
}

func (pc PeopleCounter) ID() string {
	return pc.ID_
}
func (pc PeopleCounter) Timestamp() time.Time {
	return pc.Timestamp_
}
func (pc PeopleCounter) ObjectID() string {
	return "3434"
}
func (pc PeopleCounter) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, pc.ObjectID())
}
func (pc PeopleCounter) MarshalJSON() ([]byte, error) {
	return marshalJSON(pc)
}

func NewPresence(deviceID string, digitalInputState bool, ts time.Time) Presence {
	return Presence{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		DigitalInputState: digitalInputState,
	}
}

type Presence struct {
	DeviceInfo
	DigitalInputState   bool `lwm2m:"5500"`
	DigitalInputCounter *int `lwm2m:"5501"`
}

func (d Presence) ID() string {
	return d.ID_
}
func (d Presence) Timestamp() time.Time {
	return d.Timestamp_
}
func (d Presence) ObjectID() string {
	return "3302"
}
func (d Presence) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, d.ObjectID())
}
func (d Presence) MarshalJSON() ([]byte, error) {
	return marshalJSON(d)
}

func NewDistance(deviceID string, sensorValue float64, ts time.Time) Distance {
	return Distance{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Distance struct {
	DeviceInfo
	SensorValue      float64  `lwm2m:"5700,m"`
	SensorUnits      *string  `lwm2m:"5701"`
	MinMeasuredValue *float64 `lwm2m:"5601"`
	MaxMeasuredValue *float64 `lwm2m:"5602"`
	MinRangeValue    *float64 `lwm2m:"5603"`
	MaxRangeValue    *float64 `lwm2m:"5604"`
	ApplicationType  *string  `lwm2m:"5750"`
}

func (d Distance) ID() string {
	return d.ID_
}
func (d Distance) Timestamp() time.Time {
	return d.Timestamp_
}
func (d Distance) ObjectID() string {
	return "3330"
}
func (d Distance) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, d.ObjectID())
}
func (d Distance) MarshalJSON() ([]byte, error) {
	return marshalJSON(d)
}

func NewConductivity(deviceID string, sensorValue float64, ts time.Time) Conductivity {
	return Conductivity{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Conductivity struct {
	DeviceInfo
	SensorValue float64 `lwm2m:"5700,S/m"`
	SensorUnits *string `lwm2m:"5701"`
}

func (c Conductivity) ID() string {
	return c.ID_
}
func (c Conductivity) Timestamp() time.Time {
	return c.Timestamp_
}
func (c Conductivity) ObjectID() string {
	return "3327"
}
func (c Conductivity) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, c.ObjectID())
}
func (c Conductivity) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func NewPressure(deviceID string, sensorValue float64, ts time.Time) Pressure {
	return Pressure{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Pressure struct {
	DeviceInfo
	SensorValue float64 `lwm2m:"5700,Pa"`
}

func (p Pressure) ID() string {
	return p.ID_
}
func (p Pressure) Timestamp() time.Time {
	return p.Timestamp_
}
func (p Pressure) ObjectID() string {
	return "3327"
}
func (p Pressure) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, p.ObjectID())
}
func (p Pressure) MarshalJSON() ([]byte, error) {
	return marshalJSON(p)
}

func NewPower(deviceID string, sensorValue float64, ts time.Time) Power {
	return Power{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Power struct {
	DeviceInfo
	SensorValue float64 `lwm2m:"5700,W"`
}

func (p Power) ID() string {
	return p.ID_
}
func (p Power) Timestamp() time.Time {
	return p.Timestamp_
}
func (p Power) ObjectID() string {
	return "3328"
}
func (p Power) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, p.ObjectID())
}
func (p Power) MarshalJSON() ([]byte, error) {
	return marshalJSON(p)
}

func NewEnergy(deviceID string, sensorValue float64, ts time.Time) Energy {
	return Energy{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
		SensorValue: sensorValue,
	}
}

type Energy struct {
	DeviceInfo
	SensorValue float64 `lwm2m:"5700,Wh"`
}

func (e Energy) ID() string {
	return e.ID_
}
func (e Energy) Timestamp() time.Time {
	return e.Timestamp_
}
func (e Energy) ObjectID() string {
	return "3331"
}
func (e Energy) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, e.ObjectID())
}
func (e Energy) MarshalJSON() ([]byte, error) {
	return marshalJSON(e)
}

func NewDevice(deviceID string, ts time.Time) Device {
	return Device{
		DeviceInfo: DeviceInfo{
			ID_:        deviceID,
			Timestamp_: ts,
		},
	}
}

type Device struct {
	DeviceInfo

	BatteryLevel       *int    `json:"batteryLevel" lwm2m:"9"`          //Contains the current battery level as a percentage (with a range from 0 to 100).
	PowerSourceVoltage *int    `json:"powerSourceVoltage" lwm2m:"7,mV"` //Present voltage for each Available Power Sources Resource Instance. The unit used for this resource is in mV.
	DeviceType         *string `json:"deviceType" lwm2m:"17"`           //Type of the device (manufacturer specified string: e.g. smart meters / dev Class / ...)
}

func (d Device) ID() string {
	return d.ID_
}
func (d Device) Timestamp() time.Time {
	return d.Timestamp_
}
func (d Device) ObjectID() string {
	return "3"
}
func (d Device) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, d.ObjectID())
}
func (d Device) MarshalJSON() ([]byte, error) {
	return marshalJSON(d)
}

type Stopwatch struct {
	DeviceInfo
	CumulativeTime      float64 `json:"cumulativeTime" lwm2m:"5544,s"`    //The total time in seconds that the timer input is true.
	OnOff               *bool   `json:"onOff,omitempty" lwm2m:"5850"`     //On/off control. Boolean value where True is On and False is Off
	DigitalInputCounter int32   `json:"digitalInputCounter" lwm2m:"5501"` //The cumulative value of active state detected.
}

func (d Stopwatch) ID() string {
	return d.ID_
}
func (d Stopwatch) Timestamp() time.Time {
	return d.Timestamp_
}
func (d Stopwatch) ObjectID() string {
	return "3350"
}
func (d Stopwatch) ObjectURN() string {
	return fmt.Sprintf("%s:%s", prefix, d.ObjectID())
}
func (d Stopwatch) MarshalJSON() ([]byte, error) {
	return marshalJSON(d)
}
