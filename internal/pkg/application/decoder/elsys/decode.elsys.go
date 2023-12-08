package elsys

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
