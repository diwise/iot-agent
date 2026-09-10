package application

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/application/facades"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	core "github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/senml"
	diwisepkg "github.com/diwise/senml/diwise"
	"github.com/matryer/is"
)

// Fas E: en ers-rapport (temperatur + CO2) ska med aktiverad sammanslagning
// ge exakt ett kommando med båda observationerna bevarade.
func TestMultiObjectReportSingleCommand(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", true, map[string]DeviceProfileConfig{})
	ue, _ := facades.New("servanet")(ctx, "up", []byte(ers))
	is.NoErr(agent.HandleSensorEvent(ctx, ue))

	is.Equal(len(e.SendCommandToCalls()), 1)

	cmd := e.SendCommandToCalls()[0].Command.(*core.MessageReceived)
	is.Equal(cmd.ContentType(), "application/vnd.oma.lwm2m+json")
	is.Equal(cmd.Tenant(), "")

	parsed, err := diwisepkg.Parse(cmd.Pack(), time.Now().UTC())
	is.NoErr(err)
	is.Equal(len(parsed.Objects()), 2)

	urns := map[string]bool{}
	for _, o := range parsed.Objects() {
		urns[o.URN()] = true
	}
	is.True(urns["urn:oma:lwm2m:ext:3303"])
	is.True(urns["urn:oma:lwm2m:ext:3428"])
}

// Fas E: utan aktivering bevaras ett kommando per objekt (legacy).
func TestLegacyPerObjectCommandsWithoutActivation(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", false, map[string]DeviceProfileConfig{})
	ue, _ := facades.New("servanet")(ctx, "up", []byte(ers))
	is.NoErr(agent.HandleSensorEvent(ctx, ue))

	is.Equal(len(e.SendCommandToCalls()), 2)
}

func TestValidateObservation(t *testing.T) {
	is := is.New(t)
	now := float64(time.Now().Unix())

	valid := senml.Pack{
		{BaseName: "dev/3303/", BaseTime: now, Name: "0", StringValue: "urn:oma:lwm2m:ext:3303"},
		{Name: "5700", Unit: "Cel", Value: ptr(21.5)},
	}
	is.NoErr(validateObservation(valid))

	headerOnly := senml.Pack{
		{BaseName: "dev/3303/", BaseTime: now, Name: "0", StringValue: "urn:oma:lwm2m:ext:3303"},
	}
	is.True(errors.Is(validateObservation(headerOnly), errObservationWithoutValues))

	nan := senml.Pack{
		{BaseName: "dev/3303/", BaseTime: now, Name: "0", StringValue: "urn:oma:lwm2m:ext:3303"},
		{Name: "5700", Unit: "Cel", Value: ptr(math.NaN())},
	}
	is.True(errors.Is(validateObservation(nan), errObservationNonFinite))

	inf := senml.Pack{
		{BaseName: "dev/3303/", BaseTime: now, Name: "0", StringValue: "urn:oma:lwm2m:ext:3303"},
		{Name: "5700", Unit: "Cel", Value: ptr(math.Inf(1))},
	}
	is.True(errors.Is(validateObservation(inf), errObservationNonFinite))

	boolean := senml.Pack{
		{BaseName: "dev/3200/", BaseTime: now, Name: "0", StringValue: "urn:oma:lwm2m:ext:3200"},
		{Name: "5500", BoolValue: ptrBool(true)},
	}
	is.NoErr(validateObservation(boolean))
}

// Fas E: en felaktig observation (NaN) kastas medan den giltiga skickas;
// är alla underkända skickas inget meddelande alls.
func TestReportOmitsInvalidObservations(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", true, map[string]DeviceProfileConfig{}).(*app)

	device, err := dmc.FindDeviceFromDevEUI(ctx, "a81758fffe05e6fb")
	is.NoErr(err)

	ts := time.Now().UTC()
	objects := []lwm2m.Lwm2mObject{
		lwm2m.NewTemperature("internal-id-for-device", 21.5, ts),
		lwm2m.NewTemperature("internal-id-for-device", math.NaN(), ts),
	}
	types := []string{"urn:oma:lwm2m:ext:3303"}

	is.NoErr(agent.handleReport(ctx, device, objects, types))
	is.Equal(len(e.SendCommandToCalls()), 1)

	cmd := e.SendCommandToCalls()[0].Command.(*core.MessageReceived)
	parsed, err := diwisepkg.Parse(cmd.Pack(), time.Now().UTC())
	is.NoErr(err)
	is.Equal(len(parsed.Objects()), 1)
}

func TestReportSendsNothingWhenAllInvalid(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", true, map[string]DeviceProfileConfig{}).(*app)

	device, err := dmc.FindDeviceFromDevEUI(ctx, "a81758fffe05e6fb")
	is.NoErr(err)

	ts := time.Now().UTC()
	objects := []lwm2m.Lwm2mObject{
		lwm2m.NewTemperature("internal-id-for-device", math.NaN(), ts),
	}
	types := []string{"urn:oma:lwm2m:ext:3303"}

	is.NoErr(agent.handleReport(ctx, device, objects, types))
	is.Equal(len(e.SendCommandToCalls()), 0)
}

func ptr(f float64) *float64 { return &f }

func ptrBool(b bool) *bool { return &b }
