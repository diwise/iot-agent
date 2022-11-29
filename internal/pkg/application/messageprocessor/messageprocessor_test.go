package messageprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	"github.com/diwise/iot-agent/internal/pkg/application/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/farshidtz/senml/v2"
	"github.com/matryer/is"
)

func TestProcessMessageWorksWithValidTemperatureInput(t *testing.T) {
	is, cr, ep := testSetup(t)
	mp := NewMessageReceivedProcessor(cr, ep)

	err := mp.ProcessMessage(context.Background(), newPayload(), newDevice())
	is.NoErr(err)
	is.Equal(len(ep.SendCalls()), 1) // should have been called once
}

func testSetup(t *testing.T) (*is.I, conversion.ConverterRegistry, *events.EventSenderMock) {
	is := is.New(t)

	cr := &conversion.ConverterRegistryMock{
		DesignateConvertersFunc: func(ctx context.Context, types []string) []conversion.MessageConverterFunc {
			return []conversion.MessageConverterFunc{
				func(ctx context.Context, internalID string, p payload.Payload, fn func(p senml.Pack) error) error {
					fn(senml.Pack{senml.Record{
						Name:        "0",
						StringValue: "internalID",
					}})
					return nil
				},
			}
		},
	}

	ep := &events.EventSenderMock{
		SendFunc: func(ctx context.Context, m messaging.CommandMessage) error {
			return nil
		},
		PublishFunc: func(ctx context.Context, m messaging.TopicMessage) error {
			return nil
		},
	}

	return is, cr, ep
}

func newPayload() payload.Payload {
	ts, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	p, _ := payload.New("ncaknlclkdanklcd", ts, payload.Temperature(23.5), payload.Status(0, nil))
	return p
}

func newDevice() device {
	return device{}
}

type device struct {
}

func (d device) ID() string          { return "internalID" }
func (d device) Latitude() float64   { return 0 }
func (d device) Longitude() float64  { return 0 }
func (d device) Environment() string { return "" }
func (d device) Types() []string     { return []string{""} }
func (d device) SensorType() string  { return "" }
func (d device) IsActive() bool      { return true }
func (d device) Tenant() string      { return "" }
