package x2climate

import (
	"context"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type X2ClimatePayload struct {
	//	TempC      float64 `json:"temp_c"`
	//	HumidityRH float64 `json:"humidity_rh"`
	//	VoltageMV  uint16  `json:"voltage_mv"`

	Frame        byte
	Measurement  bool
	TemperatureC float64 `json:"temp_c"`
	HumidityPct  int     `json:"humidity_rh"`
	BatteryMV    int     `json:"voltage_mv"`
	Counter      int     // bytes 10-11
	Unknown      int     // bytes 4-5

}

func (p X2ClimatePayload) BatteryLevel() *int {
	if p.BatteryMV == 0 {
		return nil
	}

	vbat := int(p.BatteryMV)
	return &vbat
}

func (p X2ClimatePayload) Error() (string, []string) {
	return "", []string{}
}

func DecoderX2Climate(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	if e.Payload.FPort != 2 {
		return nil, types.ErrInvalidFPort
	}

	p, err := decodeX2Climate(e.Payload.Data)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// decodeX2Climate decodar X2 Climate / FOAB port-2 payload.
// Förväntar sig exakt 19 bytes.
func decodeX2Climate(b []byte) (X2ClimatePayload, error) {
	var r X2ClimatePayload

	if len(b) != 19 {
		return r, types.ErrUnsupportedPayloadLength
	}

	r.Measurement = true
	r.Unknown = int(b[4])<<8 | int(b[5])
	r.BatteryMV = int(b[8])<<8 | int(b[9])
	r.Counter = int(b[10])<<8 | int(b[11])
	r.TemperatureC = float64(int16(uint16(b[12])<<8|uint16(b[13]))) / 100
	r.HumidityPct = int(b[14])

	return r, nil
}

func ConverterX2Climate(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(X2ClimatePayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts, "x2climate"), nil
}

func convertToLwm2mObjects(_ context.Context, deviceID string, p X2ClimatePayload, ts time.Time, options ...string) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	objects = append(objects, lwm2m.NewTemperature(deviceID, float64(p.TemperatureC), ts))

	objects = append(objects, lwm2m.NewHumidity(deviceID, float64(p.HumidityPct), ts))

	vdd := int(p.BatteryMV)
	d := lwm2m.NewDevice(deviceID, ts)
	d.PowerSourceVoltage = &vdd
	soc := VBatToSOC(vdd)
	d.BatteryLevel = &soc
	objects = append(objects, d)

	return objects
}

// same function as in elsys decoder
func VBatToSOC(vdd int) int {
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
