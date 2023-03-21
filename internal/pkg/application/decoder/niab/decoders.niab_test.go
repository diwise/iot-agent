package niab

import (
	"context"
	"testing"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/matryer/is"
)

func TestNIAB(t *testing.T) {
	is := is.New(t)

	evt := sensorEventFromTestData([]byte{0xcc, 0x0f, 0x03, 0xc5})
	callBackCount := 0

	FillLevelSensorDecoder(context.Background(), evt, func(ctx context.Context, p payload.Payload) error {
		battery, ok := payload.Get[int](p, payload.BatteryLevelProperty)
		is.True(ok)
		is.Equal(battery, 80)

		temperature, ok := payload.Get[float64](p, payload.TemperatureProperty)
		is.True(ok)
		is.Equal(temperature, 15.0)

		distance, ok := payload.Get[float64](p, payload.DistanceProperty)
		is.True(ok)
		is.Equal(distance, 0.965)

		callBackCount = 1
		return nil
	})

	// Make sure that the callback was in fact invoked
	is.Equal(callBackCount, 1)
}

func TestNIABCanReportMinusTemperatures(t *testing.T) {
	is := is.New(t)

	evt := sensorEventFromTestData([]byte{0xcc, 0xF0, 0x03, 0xc5})
	callBackCount := 0

	FillLevelSensorDecoder(context.Background(), evt, func(ctx context.Context, p payload.Payload) error {

		temperature, ok := payload.Get[float64](p, payload.TemperatureProperty)
		is.True(ok)
		is.Equal(temperature, -15.0)

		callBackCount = 1
		return nil
	})

	// Make sure that the callback was in fact invoked
	is.Equal(callBackCount, 1)
}

func TestNIABIgnoresReadErrors(t *testing.T) {
	is := is.New(t)

	evt := sensorEventFromTestData([]byte{0xcc, 0x0f, 0xff, 0xff})
	callBackCount := 1

	FillLevelSensorDecoder(context.Background(), evt, func(ctx context.Context, p payload.Payload) error {
		_, ok := payload.Get[int](p, payload.DistanceProperty)
		is.True(!ok)

		callBackCount = 1
		return nil
	})

	// Make sure that the callback was in fact invoked
	is.Equal(callBackCount, 1)
}

func TestNIABIgnoresTruncatedData(t *testing.T) {
	is := is.New(t)

	evt := sensorEventFromTestData([]byte{0xcc, 0x0f})
	callBackCount := 0

	err := FillLevelSensorDecoder(context.Background(), evt, func(ctx context.Context, p payload.Payload) error {
		callBackCount = 1
		return nil
	})

	// Make sure that the callback was not invoked
	is.Equal(callBackCount, 0)
	is.True(err != nil)
}

func TestNIABIgnoresTooLongData(t *testing.T) {
	is := is.New(t)

	evt := sensorEventFromTestData([]byte{0xcc, 0x0f, 0xff, 0xff, 0xff, 0xff})
	callBackCount := 0

	err := FillLevelSensorDecoder(context.Background(), evt, func(ctx context.Context, p payload.Payload) error {
		callBackCount = 1
		return nil
	})

	// Make sure that the callback was not invoked
	is.Equal(callBackCount, 0)
	is.True(err != nil)
}

func sensorEventFromTestData(data []byte) application.SensorEvent {
	return application.SensorEvent{
		Data: data,
	}
}
