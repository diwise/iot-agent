package mqtt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var tracer = otel.Tracer("iot-agent/mqtt/message-handler")

const (
	defaultForwarderQueueDepth = 256
	forwardRequestTimeout      = 15 * time.Second
)

type queuedMessage struct {
	topic     string
	payload   []byte
	qos       byte
	messageID uint16
	ack       func()
}

type messageForwarder struct {
	ctx                context.Context
	cancel             context.CancelFunc
	forwardingEndpoint string
	logger             *slog.Logger
	messageCounter     metric.Int64Counter
	httpClient         *http.Client
	jobs               chan queuedMessage
	closeOnce          sync.Once
}

func NewMessageHandler(ctx context.Context, forwardingEndpoint string) func(mqtt.Client, mqtt.Message) {
	forwarder := newMessageForwarder(ctx, forwardingEndpoint, defaultForwarderQueueDepth)
	return forwarder.Handle
}

func newMessageForwarder(ctx context.Context, forwardingEndpoint string, queueDepth int) *messageForwarder {
	messageCounter, err := otel.Meter("iot-agent/mqtt").Int64Counter(
		"diwise.mqtt.messages.total",
		metric.WithUnit("1"),
		metric.WithDescription("Total number of received mqtt messages"),
	)

	logger := logging.GetFromContext(ctx)

	if err != nil {
		logger.Error("failed to create otel message counter", "err", err.Error())
	}

	if queueDepth < 1 {
		queueDepth = 1
	}

	workerCtx, cancel := context.WithCancel(ctx)
	f := &messageForwarder{
		ctx:                workerCtx,
		cancel:             cancel,
		forwardingEndpoint: forwardingEndpoint,
		logger:             logger,
		messageCounter:     messageCounter,
		httpClient: &http.Client{
			Timeout:   forwardRequestTimeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		jobs: make(chan queuedMessage, queueDepth),
	}

	go f.run()

	return f
}

func (f *messageForwarder) Handle(client mqtt.Client, msg mqtt.Message) {
	f.messageCounter.Add(f.ctx, 1)

	job := queuedMessage{
		topic:     msg.Topic(),
		payload:   append([]byte(nil), msg.Payload()...),
		qos:       msg.Qos(),
		messageID: msg.MessageID(),
	}
	if msg.Qos() > 0 {
		job.ack = msg.Ack
	}

	select {
	case <-f.ctx.Done():
		f.logger.Warn("mqtt forwarder is shutting down; leaving message unacked", "topic", job.topic, "message_id", job.messageID)
	case f.jobs <- job:
	default:
		f.logger.Warn("mqtt forwarder queue is full; leaving message unacked", "topic", job.topic, "message_id", job.messageID)
	}
}

func (f *messageForwarder) Close() {
	f.closeOnce.Do(func() {
		f.cancel()
	})
}

func (f *messageForwarder) run() {
	for {
		select {
		case <-f.ctx.Done():
			return
		case job := <-f.jobs:
			f.forward(job)
		}
	}
}

func (f *messageForwarder) forward(job queuedMessage) {
	var err error

	ctx, span := tracer.Start(f.ctx, "forward-message")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, f.logger, ctx)

	parts := strings.Split(job.topic, "/")

	im := types.IncomingMessage{
		ID:     fmt.Sprintf("mqtt-%d", job.messageID),
		Type:   parts[len(parts)-1],
		Source: job.topic,
		Data:   job.payload,
	}

	ctx = logging.NewContextWithLogger(ctx, log, "message_id", im.ID, "received_at", time.Now().Format(time.RFC3339Nano))

	b, err := json.Marshal(im)
	if err != nil {
		log.Error("failed to marshal incoming message", "err", err.Error())
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.forwardingEndpoint, bytes.NewReader(b))
	if err != nil {
		log.Error("failed to create http request", "err", err.Error())
		return
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		log.Warn("forwarding request failed",
			"topic", job.topic,			
			"payload_snippet", string(job.payload[:min(100, len(job.payload))]),
			"payload_bytes", len(job.payload),
			"error_type", fmt.Sprintf("%T", err),
			"err", err,
		)

		return
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		ack(job)
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		ack(job)
		return
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		log.Warn("error while processing message", "topic", im.Source, "status_code", http.StatusUnprocessableEntity)
		ack(job)
		return
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
		log.Warn("dropping message after non-retryable response", "status_code", resp.StatusCode)
		ack(job)
		return
	}

	log.Error(fmt.Sprintf("unexpected response code %d", resp.StatusCode),
		"topic", job.topic,
		"payload_snippet", string(job.payload[:min(100, len(job.payload))]),
		"payload_bytes", len(job.payload))
}

func ack(job queuedMessage) {
	if job.qos > 0 && job.ack != nil {
		job.ack()
	}
}
