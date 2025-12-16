package x2climate

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/pkg/lwm2m"
)

type X2ClimatePayload struct {
	TempC      float64 `json:"temp_c"`
	HumidityRH float64 `json:"humidity_rh"`
	VoltageMV  uint16  `json:"voltage_mv"`
}

func (p X2ClimatePayload) BatteryLevel() *int {
	if p.VoltageMV == 0 {
		return nil
	}

	vbat := int(p.VoltageMV)
	return &vbat
}

func (p X2ClimatePayload) Error() (string, []string) {
	return "", []string{}
}

func DecoderX2Climate(ctx context.Context, e types.Event) (types.SensorPayload, error) {
	if e.Payload.FPort != 2 {
		return nil, fmt.Errorf("x2climate: unsupported fPort %d, expected 2", e.Payload.FPort)
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
	var t X2ClimatePayload

	if len(b) != 19 {
		return t, fmt.Errorf("invalid payload length: %d (expected 19)", len(b))
	}

	// Temperature: bytes 4–5, uint16 BE, /1600 → °C
	tempRaw := binary.BigEndian.Uint16(b[4:6])
	t.TempC = float64(tempRaw) / 1600.0

	// Humidity: bytes 8–9, uint16 BE, /100 → %RH
	humRaw := binary.BigEndian.Uint16(b[8:10])
	t.HumidityRH = float64(humRaw) / 100.0

	// Voltage: bytes 12–13, uint16 BE, already in mV
	t.VoltageMV = binary.BigEndian.Uint16(b[12:14])

	return t, nil
}

func ConverterX2Climate(ctx context.Context, deviceID string, payload types.SensorPayload, ts time.Time) ([]lwm2m.Lwm2mObject, error) {
	p := payload.(X2ClimatePayload)
	return convertToLwm2mObjects(ctx, deviceID, p, ts, "x2climate"), nil
}

func convertToLwm2mObjects(_ context.Context, deviceID string, p X2ClimatePayload, ts time.Time, options ...string) []lwm2m.Lwm2mObject {
	objects := []lwm2m.Lwm2mObject{}

	objects = append(objects, lwm2m.NewTemperature(deviceID, float64(p.TempC), ts))

	objects = append(objects, lwm2m.NewHumidity(deviceID, float64(p.HumidityRH), ts))

	vdd := int(p.VoltageMV)
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
