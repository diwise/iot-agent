// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package iotagent

import (
	"context"
	"github.com/diwise/iot-agent/internal/pkg/application"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/senml"
	"sync"
	"time"
)

// Ensure, that AppMock does implement App.
// If this is not the case, regenerate this file with moq.
var _ App = &AppMock{}

// AppMock is a mock implementation of App.
//
//	func TestSomethingThatUsesApp(t *testing.T) {
//
//		// make and configure a mocked App
//		mockedApp := &AppMock{
//			GetDeviceFunc: func(ctx context.Context, deviceID string) (dmc.Device, error) {
//				panic("mock out the GetDevice method")
//			},
//			GetMeasurementsFunc: func(ctx context.Context, deviceID string, temprel string, t time.Time, et time.Time, lastN int) ([]application.Measurement, error) {
//				panic("mock out the GetMeasurements method")
//			},
//			HandleSensorEventFunc: func(ctx context.Context, se application.SensorEvent) error {
//				panic("mock out the HandleSensorEvent method")
//			},
//			HandleSensorMeasurementListFunc: func(ctx context.Context, deviceID string, pack senml.Pack) error {
//				panic("mock out the HandleSensorMeasurementList method")
//			},
//		}
//
//		// use mockedApp in code that requires App
//		// and then make assertions.
//
//	}
type AppMock struct {
	// GetDeviceFunc mocks the GetDevice method.
	GetDeviceFunc func(ctx context.Context, deviceID string) (dmc.Device, error)

	// GetMeasurementsFunc mocks the GetMeasurements method.
	GetMeasurementsFunc func(ctx context.Context, deviceID string, temprel string, t time.Time, et time.Time, lastN int) ([]application.Measurement, error)

	// HandleSensorEventFunc mocks the HandleSensorEvent method.
	HandleSensorEventFunc func(ctx context.Context, se application.SensorEvent) error

	// HandleSensorMeasurementListFunc mocks the HandleSensorMeasurementList method.
	HandleSensorMeasurementListFunc func(ctx context.Context, deviceID string, pack senml.Pack) error

	// calls tracks calls to the methods.
	calls struct {
		// GetDevice holds details about calls to the GetDevice method.
		GetDevice []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// DeviceID is the deviceID argument value.
			DeviceID string
		}
		// GetMeasurements holds details about calls to the GetMeasurements method.
		GetMeasurements []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// DeviceID is the deviceID argument value.
			DeviceID string
			// Temprel is the temprel argument value.
			Temprel string
			// T is the t argument value.
			T time.Time
			// Et is the et argument value.
			Et time.Time
			// LastN is the lastN argument value.
			LastN int
		}
		// HandleSensorEvent holds details about calls to the HandleSensorEvent method.
		HandleSensorEvent []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Se is the se argument value.
			Se application.SensorEvent
		}
		// HandleSensorMeasurementList holds details about calls to the HandleSensorMeasurementList method.
		HandleSensorMeasurementList []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// DeviceID is the deviceID argument value.
			DeviceID string
			// Pack is the pack argument value.
			Pack senml.Pack
		}
	}
	lockGetDevice                   sync.RWMutex
	lockGetMeasurements             sync.RWMutex
	lockHandleSensorEvent           sync.RWMutex
	lockHandleSensorMeasurementList sync.RWMutex
}

// GetDevice calls GetDeviceFunc.
func (mock *AppMock) GetDevice(ctx context.Context, deviceID string) (dmc.Device, error) {
	if mock.GetDeviceFunc == nil {
		panic("AppMock.GetDeviceFunc: method is nil but App.GetDevice was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		DeviceID string
	}{
		Ctx:      ctx,
		DeviceID: deviceID,
	}
	mock.lockGetDevice.Lock()
	mock.calls.GetDevice = append(mock.calls.GetDevice, callInfo)
	mock.lockGetDevice.Unlock()
	return mock.GetDeviceFunc(ctx, deviceID)
}

// GetDeviceCalls gets all the calls that were made to GetDevice.
// Check the length with:
//
//	len(mockedApp.GetDeviceCalls())
func (mock *AppMock) GetDeviceCalls() []struct {
	Ctx      context.Context
	DeviceID string
} {
	var calls []struct {
		Ctx      context.Context
		DeviceID string
	}
	mock.lockGetDevice.RLock()
	calls = mock.calls.GetDevice
	mock.lockGetDevice.RUnlock()
	return calls
}

// GetMeasurements calls GetMeasurementsFunc.
func (mock *AppMock) GetMeasurements(ctx context.Context, deviceID string, temprel string, t time.Time, et time.Time, lastN int) ([]application.Measurement, error) {
	if mock.GetMeasurementsFunc == nil {
		panic("AppMock.GetMeasurementsFunc: method is nil but App.GetMeasurements was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		DeviceID string
		Temprel  string
		T        time.Time
		Et       time.Time
		LastN    int
	}{
		Ctx:      ctx,
		DeviceID: deviceID,
		Temprel:  temprel,
		T:        t,
		Et:       et,
		LastN:    lastN,
	}
	mock.lockGetMeasurements.Lock()
	mock.calls.GetMeasurements = append(mock.calls.GetMeasurements, callInfo)
	mock.lockGetMeasurements.Unlock()
	return mock.GetMeasurementsFunc(ctx, deviceID, temprel, t, et, lastN)
}

// GetMeasurementsCalls gets all the calls that were made to GetMeasurements.
// Check the length with:
//
//	len(mockedApp.GetMeasurementsCalls())
func (mock *AppMock) GetMeasurementsCalls() []struct {
	Ctx      context.Context
	DeviceID string
	Temprel  string
	T        time.Time
	Et       time.Time
	LastN    int
} {
	var calls []struct {
		Ctx      context.Context
		DeviceID string
		Temprel  string
		T        time.Time
		Et       time.Time
		LastN    int
	}
	mock.lockGetMeasurements.RLock()
	calls = mock.calls.GetMeasurements
	mock.lockGetMeasurements.RUnlock()
	return calls
}

// HandleSensorEvent calls HandleSensorEventFunc.
func (mock *AppMock) HandleSensorEvent(ctx context.Context, se application.SensorEvent) error {
	if mock.HandleSensorEventFunc == nil {
		panic("AppMock.HandleSensorEventFunc: method is nil but App.HandleSensorEvent was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Se  application.SensorEvent
	}{
		Ctx: ctx,
		Se:  se,
	}
	mock.lockHandleSensorEvent.Lock()
	mock.calls.HandleSensorEvent = append(mock.calls.HandleSensorEvent, callInfo)
	mock.lockHandleSensorEvent.Unlock()
	return mock.HandleSensorEventFunc(ctx, se)
}

// HandleSensorEventCalls gets all the calls that were made to HandleSensorEvent.
// Check the length with:
//
//	len(mockedApp.HandleSensorEventCalls())
func (mock *AppMock) HandleSensorEventCalls() []struct {
	Ctx context.Context
	Se  application.SensorEvent
} {
	var calls []struct {
		Ctx context.Context
		Se  application.SensorEvent
	}
	mock.lockHandleSensorEvent.RLock()
	calls = mock.calls.HandleSensorEvent
	mock.lockHandleSensorEvent.RUnlock()
	return calls
}

// HandleSensorMeasurementList calls HandleSensorMeasurementListFunc.
func (mock *AppMock) HandleSensorMeasurementList(ctx context.Context, deviceID string, pack senml.Pack) error {
	if mock.HandleSensorMeasurementListFunc == nil {
		panic("AppMock.HandleSensorMeasurementListFunc: method is nil but App.HandleSensorMeasurementList was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		DeviceID string
		Pack     senml.Pack
	}{
		Ctx:      ctx,
		DeviceID: deviceID,
		Pack:     pack,
	}
	mock.lockHandleSensorMeasurementList.Lock()
	mock.calls.HandleSensorMeasurementList = append(mock.calls.HandleSensorMeasurementList, callInfo)
	mock.lockHandleSensorMeasurementList.Unlock()
	return mock.HandleSensorMeasurementListFunc(ctx, deviceID, pack)
}

// HandleSensorMeasurementListCalls gets all the calls that were made to HandleSensorMeasurementList.
// Check the length with:
//
//	len(mockedApp.HandleSensorMeasurementListCalls())
func (mock *AppMock) HandleSensorMeasurementListCalls() []struct {
	Ctx      context.Context
	DeviceID string
	Pack     senml.Pack
} {
	var calls []struct {
		Ctx      context.Context
		DeviceID string
		Pack     senml.Pack
	}
	mock.lockHandleSensorMeasurementList.RLock()
	calls = mock.calls.HandleSensorMeasurementList
	mock.lockHandleSensorMeasurementList.RUnlock()
	return calls
}
