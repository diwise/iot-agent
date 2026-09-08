package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	iotcore "github.com/diwise/iot-core/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

// HARM-003: locks the agent -> iot-core command contract.
// The routing key, topic name and content type prefix must not change
// without a compatibility/migration plan. The payload format itself is
// owned by iot-core (pkg/messaging/events).
func TestMessageReceivedCommandContract(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{})
	ue, err := facades.New("netmore")(ctx, "payload", []byte(senlabT))
	is.NoErr(err)

	is.NoErr(agent.HandleSensorEvent(ctx, ue))

	calls := e.SendCommandToCalls()
	is.True(len(calls) > 0)

	for _, c := range calls {
		is.Equal(c.Key, "iot-core")

		m, ok := c.Command.(*iotcore.MessageReceived)
		is.True(ok)
		is.Equal(m.TopicName(), "message.received")
		is.True(strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m"))
	}
}

// BASE-010: the background cleanup goroutine must terminate on Stop,
// and Stop must be safe to call twice.
func TestAppStopIsIdempotent(t *testing.T) {
	is := is.New(t)

	agent := New(nil, nil, nil, false, "default", map[string]DeviceProfileConfig{})
	is.True(agent != nil)

	agent.Stop()
	agent.Stop()

	// Stop must also join the worker: a second Stop returns promptly
	// instead of hanging on an abandoned ticker loop.
	done := make(chan struct{})
	go func() {
		defer close(done)
		agent.Stop()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return after workers exited")
	}
}

// HARM-003: locks the device-status publication contract towards
// iot-device-management and iot-events.
func TestDeviceStatusPublicationContract(t *testing.T) {
	is, dmc, e, s, ctx := testSetup(t)

	var published []string
	e.PublishOnTopicFunc = func(_ context.Context, message messaging.TopicMessage) error {
		published = append(published, message.TopicName())
		return nil
	}

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{})
	ue, err := facades.New("netmore")(ctx, "payload", []byte(senlabT))
	is.NoErr(err)

	is.NoErr(agent.HandleSensorEvent(ctx, ue))

	is.True(len(published) > 0)
	for _, topic := range published {
		is.Equal(topic, "device-status")
	}
}
