package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/diwise/iot-agent/internal/application/facades"
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

// REV-016: locks the actual produced command body. The pack is fully
// deterministic from the fixture; the wrapper timestamp is fresh per
// message. Decoding happens through the real consumer type
// (iot-core MessageReceived), which is how iot-core will read it.
func TestMessageReceivedCommandBody(t *testing.T) {
	is := is.New(t)
	_, dmc, e, s, ctx := testSetup(t)

	agent := New(dmc, e, s, true, "default", map[string]DeviceProfileConfig{})
	ue, err := facades.New("netmore")(ctx, "payload", []byte(senlabT))
	is.NoErr(err)

	is.NoErr(agent.HandleSensorEvent(ctx, ue))

	calls := e.SendCommandToCalls()
	is.True(len(calls) > 0)

	raw := calls[0].Command.(*iotcore.MessageReceived).Body()

	// Consumer-side decode of the exact produced bytes.
	var decoded iotcore.MessageReceived
	is.NoErr(json.Unmarshal(raw, &decoded))
	is.NoErr(decoded.Error())
	is.Equal(decoded.DeviceID(), "internal-id-for-device")
	is.Equal(decoded.ObjectID(), "3303")
	is.Equal(decoded.ContentType(), "application/vnd.oma.lwm2m.ext.3303+json")
	// Tenant enrichment happens in iot-core via device lookup; the
	// agent command carries no tenant record.
	is.Equal(decoded.Tenant(), "")

	var envelope map[string]any
	is.NoErr(json.Unmarshal(raw, &envelope))

	packJSON, err := json.Marshal(envelope["pack"])
	is.NoErr(err)
	is.Equal(string(packJSON), `[{"bn":"internal-id-for-device/3303/","bt":1649740130,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":6.625}]`)

	ts, ok := envelope["timestamp"].(string)
	is.True(ok)
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	is.NoErr(err)
	is.True(time.Since(parsed) < time.Minute)
}

// BASE-010: the background cleanup goroutine must terminate on Stop,
// and Stop must be safe to call twice.
func TestAppStopIsIdempotent(t *testing.T) {
	is := is.New(t)

	agent := New(nil, nil, nil, false, "default", map[string]DeviceProfileConfig{})
	is.True(agent != nil)

	agent.Stop()
	agent.Stop()

	// Stop must join the worker: the worker's done channel closes on
	// exit, so a disabled sweep or a no-op Stop fails this test.
	impl, ok := agent.(*app)
	is.True(ok)

	select {
	case <-impl.done:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not exit after Stop")
	}

	// A second Stop after exit still returns promptly.
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
