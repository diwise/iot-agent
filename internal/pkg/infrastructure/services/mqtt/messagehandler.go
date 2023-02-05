package mqtt

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
)

var meter = global.MeterProvider().Meter("iot-agent/mqtt")
var tracer = otel.Tracer("iot-agent/mqtt/message-handler")

func NewMessageHandler(logger zerolog.Logger, forwardingEndpoint string) func(mqtt.Client, mqtt.Message) {

	messageCounter, err := meter.Int64Counter(
		"diwise.mqtt.messages.total",
		instrument.WithUnit("1"),
		instrument.WithDescription("Total number of received mqtt messages"),
	)

	if err != nil {
		logger.Error().Err(err).Msg("failed to create otel message counter")
	}

	return func(client mqtt.Client, msg mqtt.Message) {
		payload := msg.Payload()

		httpClient := http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}

		var err error

		ctx, span := tracer.Start(context.Background(), "forward-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		messageCounter.Add(ctx, 1)

		log.Debug().Str("topic", msg.Topic()).Msgf("received payload %s", string(payload))

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, forwardingEndpoint, bytes.NewBuffer(payload))
		if err != nil {
			log.Error().Err(err).Msg("failed to create http request")
			return
		}

		log.Debug().Msgf("forwarding received payload to %s", forwardingEndpoint)

		req.Header.Add("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("forwarding request failed")
		} else if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("unexpected response code %d", resp.StatusCode)
			log.Error().Err(err).Msg("failed to forward message")
		}

		msg.Ack()
	}
}
