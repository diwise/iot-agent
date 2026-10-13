package qalcosonic

import (
	"testing"

	"github.com/matryer/is"
)

// Locks the Qalcosonic status contract, including the alarm-only fix:
// Error must never panic when Reading is absent.
func TestStatusFromReading(t *testing.T) {
	is := is.New(t)

	p := Payload{
		Reading: &WaterMeterReading{
			StatusCode: NoError,
			Messages:   []string{"No error"},
		},
	}

	code, messages := p.Error()
	is.Equal(code, "0")
	is.Equal(len(messages), 0)
}

func TestStatusFromReadingWithError(t *testing.T) {
	is := is.New(t)

	p := Payload{
		Reading: &WaterMeterReading{
			StatusCode: 0x20,
			Messages:   []string{"No error", "Leak"},
		},
	}

	code, messages := p.Error()
	is.Equal(code, "32")
	is.Equal(messages, []string{"Leak"})
}

func TestStatusFromAlarmOnly(t *testing.T) {
	is := is.New(t)

	p := Payload{
		Alarms: &Alarm{
			StatusCode: 0x01,
			Messages:   []string{"Leak"},
		},
	}

	code, messages := p.Error()
	is.Equal(code, "1")
	is.Equal(messages, []string{"Leak"})
}

func TestStatusFromAlarmOnlyWithoutError(t *testing.T) {
	is := is.New(t)

	p := Payload{
		Alarms: &Alarm{
			StatusCode: NoError,
			Messages:   []string{"No error"},
		},
	}

	code, messages := p.Error()
	is.Equal(code, "0")
	is.Equal(len(messages), 0)
}

func TestStatusFromEmptyPayload(t *testing.T) {
	is := is.New(t)

	p := Payload{}

	code, messages := p.Error()
	is.Equal(code, "0")
	is.Equal(len(messages), 0)
}
