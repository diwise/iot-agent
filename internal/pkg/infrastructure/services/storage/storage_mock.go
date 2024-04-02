// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package storage

import (
	"context"
	"github.com/diwise/senml"
	"sync"
	"time"
)

// Ensure, that StorageMock does implement Storage.
// If this is not the case, regenerate this file with moq.
var _ Storage = &StorageMock{}

// StorageMock is a mock implementation of Storage.
//
//	func TestSomethingThatUsesStorage(t *testing.T) {
//
//		// make and configure a mocked Storage
//		mockedStorage := &StorageMock{
//			AddFunc: func(ctx context.Context, id string, pack senml.Pack, timestamp time.Time) error {
//				panic("mock out the Add method")
//			},
//			AddManyFunc: func(ctx context.Context, id string, packs []senml.Pack, timestamp time.Time) error {
//				panic("mock out the AddMany method")
//			},
//			GetMeasurementsFunc: func(ctx context.Context, id string, temprel string, t time.Time, et time.Time, lastN int) ([]Measurement, error) {
//				panic("mock out the GetMeasurements method")
//			},
//			InitializeFunc: func(contextMoqParam context.Context) error {
//				panic("mock out the Initialize method")
//			},
//		}
//
//		// use mockedStorage in code that requires Storage
//		// and then make assertions.
//
//	}
type StorageMock struct {
	// AddFunc mocks the Add method.
	AddFunc func(ctx context.Context, id string, pack senml.Pack, timestamp time.Time) error

	// AddManyFunc mocks the AddMany method.
	AddManyFunc func(ctx context.Context, id string, packs []senml.Pack, timestamp time.Time) error

	// GetMeasurementsFunc mocks the GetMeasurements method.
	GetMeasurementsFunc func(ctx context.Context, id string, temprel string, t time.Time, et time.Time, lastN int) ([]Measurement, error)

	// InitializeFunc mocks the Initialize method.
	InitializeFunc func(contextMoqParam context.Context) error

	// calls tracks calls to the methods.
	calls struct {
		// Add holds details about calls to the Add method.
		Add []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Pack is the pack argument value.
			Pack senml.Pack
			// Timestamp is the timestamp argument value.
			Timestamp time.Time
		}
		// AddMany holds details about calls to the AddMany method.
		AddMany []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Packs is the packs argument value.
			Packs []senml.Pack
			// Timestamp is the timestamp argument value.
			Timestamp time.Time
		}
		// GetMeasurements holds details about calls to the GetMeasurements method.
		GetMeasurements []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Temprel is the temprel argument value.
			Temprel string
			// T is the t argument value.
			T time.Time
			// Et is the et argument value.
			Et time.Time
			// LastN is the lastN argument value.
			LastN int
		}
		// Initialize holds details about calls to the Initialize method.
		Initialize []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
	}
	lockAdd             sync.RWMutex
	lockAddMany         sync.RWMutex
	lockGetMeasurements sync.RWMutex
	lockInitialize      sync.RWMutex
}

// Add calls AddFunc.
func (mock *StorageMock) Add(ctx context.Context, id string, pack senml.Pack, timestamp time.Time) error {
	if mock.AddFunc == nil {
		panic("StorageMock.AddFunc: method is nil but Storage.Add was just called")
	}
	callInfo := struct {
		Ctx       context.Context
		ID        string
		Pack      senml.Pack
		Timestamp time.Time
	}{
		Ctx:       ctx,
		ID:        id,
		Pack:      pack,
		Timestamp: timestamp,
	}
	mock.lockAdd.Lock()
	mock.calls.Add = append(mock.calls.Add, callInfo)
	mock.lockAdd.Unlock()
	return mock.AddFunc(ctx, id, pack, timestamp)
}

// AddCalls gets all the calls that were made to Add.
// Check the length with:
//
//	len(mockedStorage.AddCalls())
func (mock *StorageMock) AddCalls() []struct {
	Ctx       context.Context
	ID        string
	Pack      senml.Pack
	Timestamp time.Time
} {
	var calls []struct {
		Ctx       context.Context
		ID        string
		Pack      senml.Pack
		Timestamp time.Time
	}
	mock.lockAdd.RLock()
	calls = mock.calls.Add
	mock.lockAdd.RUnlock()
	return calls
}

// AddMany calls AddManyFunc.
func (mock *StorageMock) AddMany(ctx context.Context, id string, packs []senml.Pack, timestamp time.Time) error {
	if mock.AddManyFunc == nil {
		panic("StorageMock.AddManyFunc: method is nil but Storage.AddMany was just called")
	}
	callInfo := struct {
		Ctx       context.Context
		ID        string
		Packs     []senml.Pack
		Timestamp time.Time
	}{
		Ctx:       ctx,
		ID:        id,
		Packs:     packs,
		Timestamp: timestamp,
	}
	mock.lockAddMany.Lock()
	mock.calls.AddMany = append(mock.calls.AddMany, callInfo)
	mock.lockAddMany.Unlock()
	return mock.AddManyFunc(ctx, id, packs, timestamp)
}

// AddManyCalls gets all the calls that were made to AddMany.
// Check the length with:
//
//	len(mockedStorage.AddManyCalls())
func (mock *StorageMock) AddManyCalls() []struct {
	Ctx       context.Context
	ID        string
	Packs     []senml.Pack
	Timestamp time.Time
} {
	var calls []struct {
		Ctx       context.Context
		ID        string
		Packs     []senml.Pack
		Timestamp time.Time
	}
	mock.lockAddMany.RLock()
	calls = mock.calls.AddMany
	mock.lockAddMany.RUnlock()
	return calls
}

// GetMeasurements calls GetMeasurementsFunc.
func (mock *StorageMock) GetMeasurements(ctx context.Context, id string, temprel string, t time.Time, et time.Time, lastN int) ([]Measurement, error) {
	if mock.GetMeasurementsFunc == nil {
		panic("StorageMock.GetMeasurementsFunc: method is nil but Storage.GetMeasurements was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		ID      string
		Temprel string
		T       time.Time
		Et      time.Time
		LastN   int
	}{
		Ctx:     ctx,
		ID:      id,
		Temprel: temprel,
		T:       t,
		Et:      et,
		LastN:   lastN,
	}
	mock.lockGetMeasurements.Lock()
	mock.calls.GetMeasurements = append(mock.calls.GetMeasurements, callInfo)
	mock.lockGetMeasurements.Unlock()
	return mock.GetMeasurementsFunc(ctx, id, temprel, t, et, lastN)
}

// GetMeasurementsCalls gets all the calls that were made to GetMeasurements.
// Check the length with:
//
//	len(mockedStorage.GetMeasurementsCalls())
func (mock *StorageMock) GetMeasurementsCalls() []struct {
	Ctx     context.Context
	ID      string
	Temprel string
	T       time.Time
	Et      time.Time
	LastN   int
} {
	var calls []struct {
		Ctx     context.Context
		ID      string
		Temprel string
		T       time.Time
		Et      time.Time
		LastN   int
	}
	mock.lockGetMeasurements.RLock()
	calls = mock.calls.GetMeasurements
	mock.lockGetMeasurements.RUnlock()
	return calls
}

// Initialize calls InitializeFunc.
func (mock *StorageMock) Initialize(contextMoqParam context.Context) error {
	if mock.InitializeFunc == nil {
		panic("StorageMock.InitializeFunc: method is nil but Storage.Initialize was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
	}{
		ContextMoqParam: contextMoqParam,
	}
	mock.lockInitialize.Lock()
	mock.calls.Initialize = append(mock.calls.Initialize, callInfo)
	mock.lockInitialize.Unlock()
	return mock.InitializeFunc(contextMoqParam)
}

// InitializeCalls gets all the calls that were made to Initialize.
// Check the length with:
//
//	len(mockedStorage.InitializeCalls())
func (mock *StorageMock) InitializeCalls() []struct {
	ContextMoqParam context.Context
} {
	var calls []struct {
		ContextMoqParam context.Context
	}
	mock.lockInitialize.RLock()
	calls = mock.calls.Initialize
	mock.lockInitialize.RUnlock()
	return calls
}
