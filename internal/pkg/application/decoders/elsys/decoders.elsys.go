package elsys

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type ElsysPayload struct {
	Temperature          *float32 `json:"temperature,omitempty"`
	ExternalTemperature  *float32 `json:"externalTemperature,omitempty"`
	ExternalTemperature2 *float32 `json:"externalTemperature2,omitempty"`
	Humidity             *int8    `json:"humidity,omitempty"`

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
	SoundPeak     *uint8   `json:"soundPeak,omitempty"`
	SoundAvg      *uint8   `json:"soundAvg,omitempty"`
}

func (a ElsysPayload) BatteryLevel() *int {
	if a.VDD != nil {
		bat := int(*a.VDD)
		soc_status := MVoltToPercent(bat)
		return &soc_status
	}
	return nil
}

func (a ElsysPayload) Error() (string, []string) {
	return "", []string{}
}

func Decoder(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	var p ElsysPayload
	var err error

	if e.Payload.FPort != 5 {
		return p, types.ErrInvalidFPort
	}

	if e.Payload.Data == nil {
		return p, types.ErrPayloadContainsNoData
	}

	p, err = decodePayload(e.Payload.Data)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func Converter(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(ElsysPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts), nil
}

func ConverterEltSht3x(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(ElsysPayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts, "sht3x"), nil
}

func convertToLwm2mObjects(ctx context.Context, deviceID string, p ElsysPayload, ts time.Time, options ...string) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	if p.Temperature != nil {
		del := ""
		if p.ExternalTemperature != nil || p.ExternalTemperature2 != nil {
			del = "/0"
		}

		objects = append(objects, lwm2m.NewTemperature(deviceID+del, float64(*p.Temperature), ts))
	}

	if p.ExternalTemperature != nil {
		del := ""
		if p.Temperature != nil {
			del = "/1"
		}
		objects = append(objects, lwm2m.NewTemperature(deviceID+del, float64(*p.ExternalTemperature), ts))
	}

	if p.ExternalTemperature2 != nil {
		del := ""
		if p.Temperature != nil {
			del = "/2"
		}
		objects = append(objects, lwm2m.NewTemperature(deviceID+del, float64(*p.ExternalTemperature2), ts))
	}

	if p.Humidity != nil {
		del := ""
		if p.Pulse != nil && slices.Contains(options, "sht3x") {
			del = "/0"
		}
		objects = append(objects, lwm2m.NewHumidity(deviceID+del, float64(*p.Humidity), ts))
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
		soc := MVoltToPercent(vdd)
		d.BatteryLevel = &soc
		objects = append(objects, d)
	}

	if p.Occupancy != nil {
		objects = append(objects, lwm2m.NewPresence(deviceID, *p.Occupancy == 2, ts))
	}

	if p.SoundAvg != nil || p.SoundPeak != nil {
		soundValue := 0.0
		if p.SoundAvg != nil {
			soundValue = float64(*p.SoundAvg)
		} else {
			soundValue = float64(*p.SoundPeak)
		}

		l := lwm2m.NewLoudness(deviceID, soundValue, ts)
		if p.SoundPeak != nil {
			maxMeasuredValue := float64(*p.SoundPeak)
			l.MaxMeasuredValue = &maxMeasuredValue
		}

		objects = append(objects, l)
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

	if slices.Contains(options, "sht3x") {
		if p.Pulse != nil {
			objects = append(objects, lwm2m.NewHumidity(deviceID+"/1", float64(*p.Pulse)/10, ts))
		}
	}

	logging.GetFromContext(ctx).Debug("converted objects", slog.Int("count", len(objects)))

	return objects
}

func MVoltToPercent(vdd int) int { // till att det bara är elsys två som använder här
	halfVoltageCapacity := 3.4177
	calibrationFactor := 21.347

	batteryVolt := math.Round(float64(vdd)/10) / 100

	if batteryVolt >= 3.6 {
		batteryFull := 100.0
		return int(batteryFull)
	}

	if batteryVolt < 3.3 {
		batteryEmpty := 0.0
		return int(batteryEmpty)
	}

	soc := 100 * (1 / (1 + math.Exp(-calibrationFactor*(batteryVolt-halfVoltageCapacity))))
	roundedSoc := math.Round(soc*100) / 100
	return int(roundedSoc)
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
	require := func(i int, needed int, field string) error {
		if i+needed >= len(data) {
			return fmt.Errorf("truncated elsys payload: type 0x%02x (%s) at index %d requires %d more bytes, got %d", data[i], field, i, needed, len(data)-i-1)
		}

		return nil
	}

	neg16 := func(v int) int {
		if v > 0x7FFF {
			return -(0x010000 - v)
		}
		return v
	}

	neg8 := func(v byte) int8 {
		if v&0x80 != 0 {
			return int8(int16(v) - 0x0100)
		}

		return int8(v)
	}

	for i := 0; i < len(data); i++ {
		switch data[i] {
		case TYPE_TEMP:
			if err := require(i, 2, "temperature"); err != nil {
				return p, err
			}
			t := (int(data[i+1]) << 8) | (int(data[i+2]))
			result := float32(neg16(t)) / 10
			p.Temperature = &result
			i += 2
		case TYPE_RH:
			if err := require(i, 1, "humidity"); err != nil {
				return p, err
			}
			result := int8(int(data[i+1]))
			p.Humidity = &result
			i += 1
		case TYPE_ACC:
			if err := require(i, 3, "acceleration"); err != nil {
				return p, err
			}
			x := neg8(data[i+1])
			y := neg8(data[i+2])
			z := neg8(data[i+3])
			p.X = &x
			p.Y = &y
			p.Z = &z
			i += 3
		case TYPE_LIGHT:
			if err := require(i, 2, "light"); err != nil {
				return p, err
			}
			result := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.Light = &result
			i += 2
		case TYPE_MOTION:
			if err := require(i, 1, "motion"); err != nil {
				return p, err
			}
			result := uint8(int(data[i+1]))
			p.Motion = &result
			i += 1
		case TYPE_CO2:
			if err := require(i, 2, "co2"); err != nil {
				return p, err
			}
			result := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.CO2 = &result
			i += 2
		case TYPE_VDD:
			if err := require(i, 2, "vdd"); err != nil {
				return p, err
			}
			result := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.VDD = &result
			i += 2
		case TYPE_ANALOG1:
			if err := require(i, 2, "analog1"); err != nil {
				return p, err
			}
			a := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.Analog1 = &a
			i += 2
		case TYPE_GPS:
			if err := require(i, 6, "gps"); err != nil {
				return p, err
			}
			i += 6
		case TYPE_PULSE1:
			if err := require(i, 2, "pulse1"); err != nil {
				return p, err
			}
			pulse := uint16(int(data[i+1])<<8 | int(data[i+2]))
			p.Pulse = &pulse
			i += 2
		case TYPE_PULSE1_ABS:
			if err := require(i, 4, "pulseAbs"); err != nil {
				return p, err
			}
			pulseAbs := uint32(int(data[i+1])<<24 | int(data[i+2])<<16 | int(data[i+3])<<8 | int(data[i+4]))
			p.PulseAbs = &pulseAbs
			i += 4
		case TYPE_EXT_TEMP1:
			if err := require(i, 2, "externalTemperature"); err != nil {
				return p, err
			}
			result := float32(neg16(int(data[i+1])<<8|int(data[i+2]))) / 10
			p.ExternalTemperature = &result
			i += 2
		case TYPE_EXT_TEMP2:
			if err := require(i, 2, "externalTemperature2"); err != nil {
				return p, err
			}
			result := float32(neg16(int(data[i+1])<<8|int(data[i+2]))) / 10
			p.ExternalTemperature2 = &result
			i += 2
		case TYPE_PRESSURE:
			if err := require(i, 4, "pressure"); err != nil {
				return p, err
			}
			pressure := float32(int(data[i+1])<<24|int(data[i+2])<<16|int(data[i+3])<<8|int(data[i+4])) / 1000
			p.Pressure = &pressure
			i += 4
		case TYPE_OCCUPANCY:
			if err := require(i, 1, "occupancy"); err != nil {
				return p, err
			}
			result := uint8(int(data[i+1]))
			p.Occupancy = &result
			i += 1
		case TYPE_WATERLEAK:
			if err := require(i, 1, "waterleak"); err != nil {
				return p, err
			}
			w := uint8(int(data[i+1]))
			p.Waterleak = &w
			i += 1
		case TYPE_EXT_DIGITAL:
			if err := require(i, 1, "digital"); err != nil {
				return p, err
			}
			result := data[i+1] == 1
			p.DigitalInput = &result
			i += 1
		case TYPE_EXT_DIGITAL2:
			if err := require(i, 1, "digital2"); err != nil {
				return p, err
			}
			result := data[i+1] == 1
			p.DigitalInput2 = &result
			i += 1
		case TYPE_SOUND:
			if err := require(i, 2, "sound"); err != nil {
				return p, err
			}
			soundPeak := uint8(int(data[i+1]))
			soundAvg := uint8(int(data[i+2]))
			p.SoundPeak = &soundPeak
			p.SoundAvg = &soundAvg
			i += 2
		}
	}

	return p, nil
}
