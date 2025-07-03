package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/facades"
	"github.com/diwise/iot-agent/internal/pkg/application/types"
	"github.com/diwise/iot-agent/internal/pkg/presentation/api/auth"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/net/http/router"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("iot-agent/api")

func RegisterHandlers(ctx context.Context, rootMux *http.ServeMux, app application.App, facade facades.EventFunc, policies io.Reader) error {
	const apiPrefix string = "/api/v0"

	authenticator, err := auth.NewAuthenticator(ctx, policies)
	if err != nil {
		return fmt.Errorf("failed to create api authenticator: %w", err)
	}

	r := router.New(rootMux, router.WithPrefix(apiPrefix))
	r.Group(func(sm router.ServeMux) {
		sm.Use(authenticator)
		
		sm.Post("/messages", NewIncomingMessageHandler(ctx, app, facade))
		sm.Post("/messages/lwm2m", NewIncomingLWM2MMessageHandler(ctx, app))
	})

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
	logger := logging.GetFromContext(ctx)

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
