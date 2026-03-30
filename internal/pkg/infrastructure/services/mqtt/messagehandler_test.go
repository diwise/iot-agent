package mqtt

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type fakeMessage struct {
	topic   string
	payload []byte
	qos     byte
	acked   atomic.Int32
}

func (m *fakeMessage) Duplicate() bool   { return false }
func (m *fakeMessage) Qos() byte         { return m.qos }
func (m *fakeMessage) Retained() bool    { return false }
func (m *fakeMessage) Topic() string     { return m.topic }
func (m *fakeMessage) MessageID() uint16 { return 1 }
func (m *fakeMessage) Payload() []byte   { return m.payload }
func (m *fakeMessage) Ack()              { m.acked.Add(1) }

func TestMessageHandlerAcksOnCreated(t *testing.T) {
	ctx := t.Context()

	var requestCount atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer s.Close()

	h := NewMessageHandler(ctx, s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	waitFor(t, func() bool { return requestCount.Load() == 1 })
	waitFor(t, func() bool { return msg.acked.Load() == 1 })
}

func TestMessageHandlerDoesNotAckOnServerError(t *testing.T) {
	ctx := t.Context()

	var requestCount atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s.Close()

	h := NewMessageHandler(ctx, s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	waitFor(t, func() bool { return requestCount.Load() == 1 })
	time.Sleep(20 * time.Millisecond)
	if msg.acked.Load() != 0 {
		t.Fatalf("expected ack count 0, got %d", msg.acked.Load())
	}
}

func TestMessageHandlerAcksOnClientError(t *testing.T) {
	ctx := t.Context()

	var requestCount atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer s.Close()

	h := NewMessageHandler(ctx, s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	waitFor(t, func() bool { return requestCount.Load() == 1 })
	waitFor(t, func() bool { return msg.acked.Load() == 1 })
}

func TestMessageHandlerAcksOnNotFound(t *testing.T) {
	ctx := t.Context()

	var requestCount atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	h := NewMessageHandler(ctx, s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	waitFor(t, func() bool { return requestCount.Load() == 1 })
	waitFor(t, func() bool { return msg.acked.Load() == 1 })
}

func TestMessageHandlerAcksOnUnprocessableEntity(t *testing.T) {
	ctx := t.Context()

	var requestCount atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer s.Close()

	h := NewMessageHandler(ctx, s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	waitFor(t, func() bool { return requestCount.Load() == 1 })
	waitFor(t, func() bool { return msg.acked.Load() == 1 })
}

func TestMessageHandlerReturnsBeforeSlowForwardCompletes(t *testing.T) {
	ctx := t.Context()

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-releaseRequest
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer s.Close()

	h := NewMessageHandler(ctx, s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	returned := make(chan struct{})
	go func() {
		h(nil, msg)
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected message handler to return quickly without waiting for HTTP forwarding")
	}

	select {
	case <-requestStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("expected worker to start forwarding the queued message")
	}

	if msg.acked.Load() != 0 {
		t.Fatalf("expected message to remain unacked while forwarding is in progress, got %d", msg.acked.Load())
	}

	close(releaseRequest)
	waitFor(t, func() bool { return msg.acked.Load() == 1 })
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}
