package elsys

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
)

func Decoder(ctx context.Context, ue application.SensorEvent, fn func(context.Context, payload.Payload) error) error {
	d := struct {
		Temperature         *float32 `json:"temperature,omitempty"`
		ExternalTemperature *float32 `json:"externalTemperature,omitempty"`
		Vdd                 *int     `json:"vdd,omitempty"`
		CO2                 *int     `json:"co2,omitempty"`
		Humidity            *float32 `json:"humidity,omitempty"`
		Light               *int     `json:"light,omitempty"`
		Motion              *int     `json:"motion,omitempty"`
		Occupancy           *int     `json:"occupancy,omitempty"`
		DigitalInput        *int     `json:"digital"`
		DigitalInputCounter *int64   `json:"pulseAbs"`
	}{}

	if ue.Object == nil {
		elsysPayload := DecodeElsysPayload(ue.Data)

		var decorators []payload.PayloadDecoratorFunc

		if d.Temperature != nil {
			decorators = append(decorators, payload.Temperature(float64(elsysPayload.Temperature)))
		}

		if d.ExternalTemperature != nil {
			decorators = append(decorators, payload.Temperature(float64(elsysPayload.ExternalTemperature)))
		}

		if d.CO2 != nil {
			decorators = append(decorators, payload.CO2(int(elsysPayload.CO2)))
		}

		if d.Humidity != nil {
			decorators = append(decorators, payload.Humidity(float32(elsysPayload.Humidity)))
		}

		if d.Light != nil {
			decorators = append(decorators, payload.Light(int(elsysPayload.Light)))
		}

		if d.Motion != nil {
			decorators = append(decorators, payload.Motion(int(elsysPayload.Motion)))
		}

		if d.Vdd != nil {
			decorators = append(decorators, payload.BatteryLevel(int(elsysPayload.VDD)))
		}

		if d.Occupancy != nil {
			// 0 = Unoccupied / 1 = Pending (Entering or leaving) / 2 = Occupied
			decorators = append(decorators, payload.Presence(elsysPayload.Occupancy == 2))
		}

		if d.DigitalInput != nil {
			decorators = append(decorators, payload.DigitalInputState(elsysPayload.DigitalInput))
		}

		if p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...); err == nil {
			return fn(ctx, p)
		} else {
			return err
		}

	}

	err := json.Unmarshal(ue.Object, &d)
	if err != nil {
		return fmt.Errorf("failed to unmarshal elsys payload: %s", err.Error())
	}

	var decorators []payload.PayloadDecoratorFunc

	if d.Temperature != nil {
		decorators = append(decorators, payload.Temperature(float64(*d.Temperature)))
	}

	if d.ExternalTemperature != nil {
		decorators = append(decorators, payload.Temperature(float64(*d.ExternalTemperature)))
	}

	if d.CO2 != nil {
		decorators = append(decorators, payload.CO2(*d.CO2))
	}

	if d.Humidity != nil {
		decorators = append(decorators, payload.Humidity(*d.Humidity))
	}

	if d.Light != nil {
		decorators = append(decorators, payload.Light(*d.Light))
	}

	if d.Motion != nil {
		decorators = append(decorators, payload.Motion(*d.Motion))
	}

	if d.Vdd != nil {
		decorators = append(decorators, payload.BatteryLevel(*d.Vdd))
	}

	if d.Occupancy != nil {
		// 0 = Unoccupied / 1 = Pending (Entering or leaving) / 2 = Occupied
		decorators = append(decorators, payload.Presence(*d.Occupancy == 2))
	}

	if d.DigitalInput != nil {
		decorators = append(decorators, payload.DigitalInputState(*d.DigitalInput == 1))
	}

	if d.DigitalInputCounter != nil {
		decorators = append(decorators, payload.DigitalInputCounter(*d.DigitalInputCounter))
	}

	if p, err := payload.New(ue.DevEui, ue.Timestamp, decorators...); err == nil {
		return fn(ctx, p)
	} else {
		return err
	}
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

type ElsysPayload struct {
	Temperature         float32 `json:"temperature"`
	ExternalTemperature float32 `json:"externalTemperature"`
	Humidity            int8    `json:"humidity"`

	//Acceleration
	X int8 `json:"x"`
	Y int8 `json:"y"`
	Z int8 `json:"z"`

	Light   uint16 `json:"light"`
	Motion  uint8  `json:"motion"`
	CO2     uint16 `json:"co2"`
	VDD     uint16 `json:"vdd"`
	Analog1 uint16 `json:"analog1"`

	//GPS
	Lat float32 `json:"lat"`
	Lon float32 `json:"long"`

	Pulse         uint16  `json:"pulse1"`
	PulseAbs      uint32  `json:"pulseAbs"`
	Pressure      float32 `json:"pressure"`
	Occupancy     uint8   `json:"occupancy"`
	DigitalInput  bool    `json:"digital"`
	DigitalInput2 bool    `json:"digital2"`
	Waterleak     uint8   `json:"waterleak"`
}

func DecodeElsysPayload(data []byte) ElsysPayload {

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
			p.Temperature = float32(neg16(t)) / 10
			i += 2
		case TYPE_RH:
			p.Humidity = int8(int(data[i+1]))
			i += 1
		case TYPE_ACC:
			p.X = neg8(int8(int(data[i+1])))
			p.Y = neg8(int8(int(data[i+2])))
			p.Z = neg8(int8(int(data[i+3])))
			i += 3
		case TYPE_LIGHT:
			p.Light = uint16(int(data[i+1])<<8 | int(data[i+2]))
			i += 2
		case TYPE_MOTION:
			p.Motion = uint8(int(data[i+1]))
			i += 1
		case TYPE_CO2:
			p.CO2 = uint16(int(data[i+1])<<8 | int(data[i+2]))
			i += 2
		case TYPE_VDD:
			p.VDD = uint16(int(data[i+1])<<8 | int(data[i+2]))
			i += 2
		case TYPE_ANALOG1:
			p.Analog1 = uint16(int(data[i+1])<<8 | int(data[i+2]))
			i += 2
		case TYPE_GPS:
			i += 6
		case TYPE_PULSE1:
			p.Pulse = uint16(int(data[i+1])<<8 | int(data[i+2]))
			i += 2
		case TYPE_PULSE1_ABS:
			p.PulseAbs = uint32(int(data[i+1])<<24 | int(data[i+2])<<16 | int(data[i+3])<<8 | int(data[i+4]))
			i += 4
		case TYPE_EXT_TEMP1:
			p.ExternalTemperature = float32(neg16(int(data[i+1])<<8|int(data[i+2]))) / 10
			i += 2
		case TYPE_PRESSURE:
			p.Pressure = float32(int(data[i+1])<<24|int(data[i+2])<<16|int(data[i+3])<<8|int(data[i+4])) / 1000
			i += 4
		case TYPE_OCCUPANCY:
			p.Occupancy = uint8(int(data[i+1]))
			i += 1
		case TYPE_WATERLEAK:
			p.Waterleak = uint8(int(data[i+1]))
			i += 1
		case TYPE_EXT_DIGITAL:
			p.DigitalInput = data[i+1] == 1
			i += 1
		case TYPE_EXT_DIGITAL2:
			p.DigitalInput2 = data[i+1] == 1
			i += 1
		}
	}

	return p
}
