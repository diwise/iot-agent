package mqtt

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
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
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer s.Close()

	h := NewMessageHandler(context.Background(), s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	if msg.acked.Load() != 1 {
		t.Fatalf("expected ack count 1, got %d", msg.acked.Load())
	}
}

func TestMessageHandlerDoesNotAckOnServerError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s.Close()

	h := NewMessageHandler(context.Background(), s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	if msg.acked.Load() != 0 {
		t.Fatalf("expected ack count 0, got %d", msg.acked.Load())
	}
}

func TestMessageHandlerAcksOnClientError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer s.Close()

	h := NewMessageHandler(context.Background(), s.URL)
	msg := &fakeMessage{topic: "a/b/up", payload: []byte(`{"k":"v"}`), qos: 1}

	h(nil, msg)

	if msg.acked.Load() != 1 {
		t.Fatalf("expected ack count 1, got %d", msg.acked.Load())
	}
}
