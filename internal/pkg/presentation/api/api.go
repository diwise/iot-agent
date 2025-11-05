package api

import (
	"context"
	"encoding/json"
	"errors"

	//"fmt"
	"io"
	"net/http"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"

	//"github.com/diwise/iot-agent/internal/pkg/presentation/api/auth"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/net/http/router"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var tracer = otel.Tracer("iot-agent/api")

func RegisterHandlers(ctx context.Context, rootMux *http.ServeMux, app application.App, facade facades.EventFunc) error {
	const apiPrefix string = "/api/v0"

	r := router.New(rootMux, router.WithPrefix(apiPrefix), router.WithTaggedRoutes(true))

	r.Post("/messages", NewIncomingMessageHandler(ctx, app, facade))
	r.Post("/messages/lwm2m", NewIncomingLWM2MMessageHandler(ctx, app))

	return nil
}

func NewIncomingMessageHandler(ctx context.Context, app application.App, facade facades.EventFunc) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		b, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var im types.IncomingMessage
		err = json.Unmarshal(b, &im)
		if err != nil {
			log.Error("failed to unmarshal incoming message", "err", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		evt, err := facade(ctx, im.Type, im.Data)
		if err != nil {
			log.Error("failed to decode sensor event using facade", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if evt.DevEUI == "" {
			log.Debug("could not handle message", "type", im.Type, "reason", "DevEUI is missing")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// add source to event to use for auto create devices (TODO)
		evt.Source = im.Source

		err = app.HandleSensorEvent(ctx, evt)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func NewIncomingLWM2MMessageHandler(ctx context.Context, app application.App) http.HandlerFunc {
	messageCounter, err := otel.Meter("iot-agent/lwm2m").Int64Counter(
		"diwise.lwm2m.messages.total",
		metric.WithUnit("1"),
		metric.WithDescription("Total number of received lwm2m messages"),
	)

	logger := logging.GetFromContext(ctx)

	if err != nil {
		logger.Error("failed to create otel message counter", "err", err.Error())
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-lwm2m-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		msg, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		pack := senml.Pack{}

		err = json.Unmarshal(msg, &pack)
		if err == nil && len(pack) == 0 {
			err = errors.New("empty senML pack received")
		}

		if err != nil {
			log.Error("failed to decode incoming senML pack", "err", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		messageCounter.Add(ctx, 1)

		deviceID := lwm2m.DeviceID(pack)
		err = app.HandleSensorMeasurementList(ctx, deviceID, pack)

		if err != nil {
			log.Error("failed to handle measurement list", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
