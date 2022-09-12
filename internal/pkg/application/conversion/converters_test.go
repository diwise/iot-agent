package conversion

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/matryer/is"
)

func TestThatTemperatureDecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	payload := decoder.Payload{
		DevEUI:    "ncaknlclkdanklcd",
		Timestamp: "2006-01-02T15:04:05Z",
	}
	temp := struct {
		Temperature float32 `json:"temperature"`
	}{
		22.2,
	}
	payload.Measurements = append(payload.Measurements, temp)

	msg, err := Temperature(ctx, "internalID", payload)

	is.NoErr(err)
	is.Equal(22.2, *msg[1].Value)
}

func TestThatCO2DecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	payload := decoder.Payload{
		DevEUI:    "ncaknlclkdanklcd",
		Timestamp: "2006-01-02T15:04:05Z",
	}
	co2 := struct {
		CO2 int `json:"co2"`
	}{
		22,
	}
	payload.Measurements = append(payload.Measurements, co2)

	msg, err := AirQuality(ctx, "internalID", payload)

	is.NoErr(err)
	is.Equal(22.0, *msg[1].Value)
}

func TestThatPresenceDecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	payload := decoder.Payload{
		DevEUI:    "ncaknlclkdanklcd",
		Timestamp: "2006-01-02T15:04:05Z",
	}
	present := struct {
		Presence bool `json:"present"`
	}{
		true,
	}
	payload.Measurements = append(payload.Measurements, present)

	msg, err := Presence(ctx, "internalID", payload)

	is.NoErr(err)
	is.True(*msg[1].BoolValue)
}

func TestThatWatermeterDecodesValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	payload := decoder.Payload{
		DevEUI:     "3489573498573459",
		DeviceName: "deviceName",
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	curDateTime := struct {
		CurrentDateTime string `json:"currentTime"`
	}{
		"2006-01-02T15:04:05Z",
	}
	payload.Measurements = append(payload.Measurements, curDateTime)
	curVol := struct {
		CurrentVolume float64 `json:"currentVolume"`
	}{
		1009,
	}
	payload.Measurements = append(payload.Measurements, curVol)

	msg, err := Watermeter(ctx, "internalID", payload)

	is.NoErr(err)
	is.True(msg != nil)

	is.Equal(msg[1].Name, "DeviceName")
	is.Equal(msg[1].StringValue, "deviceName")

	is.Equal(msg[2].Name, "CurrentDateTime")
	is.Equal(msg[2].StringValue, "2006-01-02T15:04:05Z")

	is.Equal(msg[3].Name, "CumulatedWaterVolume")
	is.Equal(*msg[3].Value, 1009.0)
}

func mcmTestSetup(t *testing.T) (*is.I, context.Context) {
	ctx, _ := logging.NewLogger(context.Background(), "test", "")
	return is.New(t), ctx
}
