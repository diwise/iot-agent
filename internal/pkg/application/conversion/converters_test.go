package conversion

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/matryer/is"
)

func TestThatTemperatureDecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Temperature(22.2))

	msg, err := Temperature(ctx, "internalID", p)

	is.NoErr(err)
	is.Equal(float64(22.200000762939453), *msg[1].Value)
}

func TestThatCO2DecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.CO2(22))

	msg, err := AirQuality(ctx, "internalID", p)

	is.NoErr(err)
	is.Equal(float64(22), *msg[1].Value)
}

func TestThatPresenceDecodesValueCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)
	p, _ := payload.New("ncaknlclkdanklcd", toT("2006-01-02T15:04:05Z"), payload.Presence(true))

	msg, err := Presence(ctx, "internalID", p)

	is.NoErr(err)
	is.True(*msg[1].BoolValue)
}

func TestThatWatermeterDecodesValuesCorrectly(t *testing.T) {
	is, ctx := mcmTestSetup(t)

	p, _ := payload.New("3489573498573459", time.Now(),
		payload.CurrentTime(toT("2006-01-02T15:04:05Z")),
		payload.CurrentVolume(1009),
		payload.DeltaVolume(100, 2000, toT("2020-09-09T12:32:21Z")))

	msg, err := Watermeter(ctx, "internalID", p)

	is.NoErr(err)
	is.True(msg != nil)

	is.Equal(msg[2].Name, "CurrentDateTime")
	is.Equal(msg[2].StringValue, "2006-01-02T15:04:05Z")

	is.Equal(msg[1].Name, "CumulatedWaterVolume")
	is.Equal(*msg[1].Value, 1009.0)
}

func mcmTestSetup(t *testing.T) (*is.I, context.Context) {
	ctx, _ := logging.NewLogger(context.Background(), "test", "")
	return is.New(t), ctx
}

func toT(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	} else {
		panic(err)
	}
}
