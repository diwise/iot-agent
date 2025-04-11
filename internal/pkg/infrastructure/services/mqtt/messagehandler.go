package mqtt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var tracer = otel.Tracer("iot-agent/mqtt/message-handler")

func NewMessageHandler(ctx context.Context, forwardingEndpoint string) func(mqtt.Client, mqtt.Message) {

	messageCounter, err := otel.Meter("iot-agent/mqtt").Int64Counter(
		"diwise.mqtt.messages.total",
		metric.WithUnit("1"),
		metric.WithDescription("Total number of received mqtt messages"),
	)

	logger := logging.GetFromContext(ctx)

	if err != nil {
		logger.Error("failed to create otel message counter", "err", err.Error())
	}

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	return func(client mqtt.Client, msg mqtt.Message) {
		go func() {
			var err error

			ctx, span := tracer.Start(context.Background(), "forward-message")
			defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
			_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

			messageCounter.Add(ctx, 1)

			defer msg.Ack()
			payload := msg.Payload()

			parts := strings.Split(msg.Topic(), "/")

			params := url.Values{}
			params.Add("type", parts[len(parts)-1])
			params.Add("source", msg.Topic())

			endpoint := fmt.Sprintf("%s?%s", forwardingEndpoint, params.Encode())

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(payload))
			if err != nil {
				log.Error("failed to create http request", "err", err.Error())
				return
			}

			log.Debug("forwarding received payload", "topic", msg.Topic(), "endpoint", endpoint)

			req.Header.Add("Content-Type", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				log.Error("forwarding request failed", "err", err.Error())
			} else {
				defer func() {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}()

				if resp.StatusCode != http.StatusCreated {
					err = fmt.Errorf("unexpected response code %d", resp.StatusCode)
					log.Error("failed to forward message", "err", err.Error())
				}
			}
		}()
	}
}
