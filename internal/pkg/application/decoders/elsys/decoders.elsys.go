package elsys

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type ElsysPayload struct {
	Temperature         *float32 `json:"temperature,omitempty"`
	ExternalTemperature *float32 `json:"externalTemperature,omitempty"`
	Humidity            *int8    `json:"humidity,omitempty"`

	//Acceleration
	X *int8 `json:"x,omitempty"`
	Y *int8 `json:"y,omitempty"`
	Z *int8 `json:"z,omitempty"`

	Light   *uint16 `json:"light,omitempty"`
	Motion  *uint8  `json:"motion,omitempty"`
	CO2     *uint16 `json:"co2,omitempty"`
	VDD     *uint16 `json:"vdd,omitempty"`
	Analog1 *uint16 `json:"analog1,omitempty"`

	//GPS
	Lat *float32 `json:"lat,omitempty"`
	Lon *float32 `json:"long,omitempty"`

	Pulse         *uint16  `json:"pulse1,omitempty"`
	PulseAbs      *uint32  `json:"pulseAbs,omitempty"`
	Pressure      *float32 `json:"pressure,omitempty"`
	Occupancy     *uint8   `json:"occupancy,omitempty"`
	DigitalInput  *bool    `json:"digital,omitempty"`
	DigitalInput2 *bool    `json:"digital2,omitempty"`
	Waterleak     *uint8   `json:"waterleak,omitempty"`
}

func Decoder(ctx context.Context, e types.SensorEvent) (any, error) {
	var p ElsysPayload
	var err error

	if e.Object != nil {
		obj := struct {
			Temperature         *float32 `json:"temperature,omitempty"`
			ExternalTemperature *float32 `json:"externalTemperature,omitempty"`
			Humidity            *int8    `json:"humidity,omitempty"`
			Light               *uint16  `json:"light,omitempty"`
			Motion              *uint8   `json:"motion,omitempty"`
			CO2                 *uint16  `json:"co2,omitempty"`
			VDD                 *uint16  `json:"vdd,omitempty"`
			Analog1             *uint16  `json:"analog1,omitempty"`
			Pulse               *uint16  `json:"pulse1,omitempty"`
			PulseAbs            *uint32  `json:"pulseAbs,omitempty"`
			Pressure            *float32 `json:"pressure,omitempty"`
			Occupancy           *uint8   `json:"occupancy,omitempty"`
			DigitalInput        *int     `json:"digital,omitempty"`
			DigitalInput2       *int     `json:"digital2,omitempty"`
			Waterleak           *uint8   `json:"waterleak,omitempty"`
		}{}
		err := json.Unmarshal(e.Object, &obj)
		if err != nil {
			return nil, err
		}

		p.Temperature = obj.Temperature
		p.ExternalTemperature = obj.ExternalTemperature
		p.Humidity = obj.Humidity
		p.Light = obj.Light
		p.Motion = obj.Motion
		p.CO2 = obj.CO2
		p.VDD = obj.VDD
		p.Analog1 = obj.Analog1
		p.Pulse = obj.Pulse
		p.PulseAbs = obj.PulseAbs
		p.Pressure = obj.Pressure
		p.Occupancy = obj.Occupancy
		if obj.DigitalInput != nil {
			b := *obj.DigitalInput == 1
			p.DigitalInput = &b
		}
		if obj.DigitalInput2 != nil {
			b := *obj.DigitalInput2 == 1
			p.DigitalInput2 = &b
		}
		p.Waterleak = obj.Waterleak
	} else {
		p, err = decodePayload(e.Data)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func Converter(ctx context.Context, deviceID string, payload any, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(ElsysPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p ElsysPayload, ts time.Time) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Temperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, float64(*p.Temperature), ts))
	}

	if p.ExternalTemperature != nil {
		objects = append(objects, lwm2m.NewTemperature(deviceID, float64(*p.ExternalTemperature), ts))
	}

	if p.Humidity != nil {
		objects = append(objects, lwm2m.NewHumidity(deviceID, float64(*p.Humidity), ts))
	}

	if p.Light != nil {
		objects = append(objects, lwm2m.NewIlluminance(deviceID, float64(*p.Light), ts))
	}

	if p.CO2 != nil {
		co2 := float64(*p.CO2)
		objects = append(objects, lwm2m.NewAirQuality(deviceID, &co2, nil, nil, nil, ts))
	}

	if p.VDD != nil {
		vdd := int(*p.VDD)
		d := lwm2m.NewDevice(deviceID, ts)
		d.PowerSourceVoltage = &vdd
		objects = append(objects, d)
	}

	if p.Occupancy != nil {
		objects = append(objects, lwm2m.NewPresence(deviceID, *p.Occupancy == 2, ts))
	}

	if p.DigitalInput != nil {
		var pulseAbs *int
		if p.PulseAbs != nil {
			pulseAbs = new(int)
			*pulseAbs = int(*p.PulseAbs)
		}

		di := lwm2m.NewDigitalInput(deviceID, *p.DigitalInput, ts)
		di.DigitalInputCounter = pulseAbs

		objects = append(objects, di)
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

const (
	TYPE_TEMP          = 0x01 //temp 2 bytes -3276.8°C -->3276.7°C
	TYPE_RH            = 0x02 //Humidity 1 byte  0-100%
	TYPE_ACC           = 0x03 //acceleration 3 bytes X,Y,Z -128 --> 127 +/-63=1G
	TYPE_LIGHT         = 0x04 //Light 2 bytes 0-->65535 Lux
	TYPE_MOTION        = 0x05 //No of motion 1 byte  0-255
	TYPE_CO2           = 0x06 //Co2 2 bytes 0-65535 ppm
	TYPE_VDD           = 0x07 //VDD 2byte 0-65535mV
	TYPE_ANALOG1       = 0x08 //VDD 2byte 0-65535mV
	TYPE_GPS           = 0x09 //3bytes lat 3bytes long binary
	TYPE_PULSE1        = 0x0A //2bytes relative pulse count
	TYPE_PULSE1_ABS    = 0x0B //4bytes no 0->0xFFFFFFFF
	TYPE_EXT_TEMP1     = 0x0C //2bytes -3276.5C-->3276.5C
	TYPE_EXT_DIGITAL   = 0x0D //1bytes value 1 or 0
	TYPE_EXT_DISTANCE  = 0x0E //2bytes distance in mm
	TYPE_ACC_MOTION    = 0x0F //1byte number of vibration/motion
	TYPE_IR_TEMP       = 0x10 //2bytes internal temp 2bytes external temp -3276.5C-->3276.5C
	TYPE_OCCUPANCY     = 0x11 //1byte data
	TYPE_WATERLEAK     = 0x12 //1byte data 0-255
	TYPE_GRIDEYE       = 0x13 //65byte temperature data 1byte ref+64byte external temp
	TYPE_PRESSURE      = 0x14 //4byte pressure data (hPa)
	TYPE_SOUND         = 0x15 //2byte sound data (peak/avg)
	TYPE_PULSE2        = 0x16 //2bytes 0-->0xFFFF
	TYPE_PULSE2_ABS    = 0x17 //4bytes no 0->0xFFFFFFFF
	TYPE_ANALOG2       = 0x18 //2bytes voltage in mV
	TYPE_EXT_TEMP2     = 0x19 //2bytes -3276.5C-->3276.5C
	TYPE_EXT_DIGITAL2  = 0x1A // 1bytes value 1 or 0
	TYPE_EXT_ANALOG_UV = 0x1B // 4 bytes signed int (uV)
	TYPE_TVOC          = 0x1C // 2 bytes (ppb)
	TYPE_DEBUG         = 0x3D // 4bytes debug
)

func decodePayload(data []byte) (ElsysPayload, error) {
	p := ElsysPayload{}

	neg16 := func(v int) int {
		if v > 0x7FFF {
			return -(0x010000 - v)
		}
		return v
	}

	neg8 := func(v int8) int8 {
		if v > 0x7F {
			return int8(-(0x0100 - int16(v)))
		}
		return int8(v)
	}

	for i := 0; i < len(data); i++ {
		switch data[i] {
		case TYPE_TEMP:
			t := (int(data[i+1]) << 8) | (int(data[i+2]))
			result := float32(neg16(t)) / 10
			p.Temperature = &result
			i += 2
		case TYPE_RH:
			result := int8(int(data[i+1]))
			p.Humidity = &result
			i += 1
		case TYPE_ACC:
			x := neg8(int8(int(data[i+1])))
			y := neg8(int8(int(data[i+2])))
			z := neg8(int8(int(data[i+3])))
			p.X = &x
			p.Y = &y
			p.Z = &z
			i += 3
		case TYPE_LIGHT:
			result := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.Light = &result
			i += 2
		case TYPE_MOTION:
			result := uint8(int(data[i+1]))
			p.Motion = &result
			i += 1
		case TYPE_CO2:
			result := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.CO2 = &result
			i += 2
		case TYPE_VDD:
			result := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.VDD = &result
			i += 2
		case TYPE_ANALOG1:
			a := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.Analog1 = &a
			i += 2
		case TYPE_GPS:
			i += 6
		case TYPE_PULSE1:
			pulse := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.Pulse = &pulse
			i += 2
		case TYPE_PULSE1_ABS:
			pulseAbs := uint32(int(data[i+1])<<24 | int(data[i+2])<<16 | int(data[i+3])<<8 | int(data[i+4]))
			p.PulseAbs = &pulseAbs
			i += 4
		case TYPE_EXT_TEMP1:
			result := float32(neg16(int(data[i+1])<<8|int(data[i+2]))) / 10
			p.ExternalTemperature = &result
			i += 2
		case TYPE_PRESSURE:
			pressure := float32(int(data[i+1])<<24|int(data[i+2])<<16|int(data[i+3])<<8|int(data[i+4])) / 1000
			p.Pressure = &pressure
			i += 4
		case TYPE_OCCUPANCY:
			result := uint8(int(data[i+1]))
			p.Occupancy = &result
			i += 1
		case TYPE_WATERLEAK:
			w := uint8(int(data[i+1]))
			p.Waterleak = &w
			i += 1
		case TYPE_EXT_DIGITAL:
			result := data[i+1] == 1
			p.DigitalInput = &result
			i += 1
		case TYPE_EXT_DIGITAL2:
			result := data[i+1] == 1
			p.DigitalInput2 = &result
			i += 1
		}
	}

	return p, nil
}
