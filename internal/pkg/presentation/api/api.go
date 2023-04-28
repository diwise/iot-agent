package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/farshidtz/senml/v2"
	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/rs/cors"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("iot-agent/api")

type API interface {
	Router() chi.Router
	health(w http.ResponseWriter, r *http.Request)
}

type api struct {
	r                  chi.Router
	app                iotagent.App
	forwardingEndpoint string
}

func New(ctx context.Context, r chi.Router, facade, forwardingEndpoint string, app iotagent.App) API {
	return newAPI(ctx, r, facade, forwardingEndpoint, app)
}

func newAPI(ctx context.Context, r chi.Router, facade, forwardingEndpoint string, app iotagent.App) *api {

	a := &api{
		r:                  r,
		app:                app,
		forwardingEndpoint: forwardingEndpoint,
	}

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	serviceName := "iot-agent"

	r.Use(otelchi.Middleware(serviceName, otelchi.WithChiRoutes(r)))

	r.Get("/health", a.health)
	r.Post("/api/v0/messages", a.incomingMessageHandler(ctx, facade))
	r.Post("/api/v0/messages/lwm2m", a.incomingLWM2MMessageHandler(ctx))
	r.Post("/api/v0/messages/schneider", a.incomingSchneiderMessageHandler(ctx))

	return a
}

func (a *api) Router() chi.Router {
	return a.r
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) incomingMessageHandler(ctx context.Context, defaultFacade string) http.HandlerFunc {
	facade := application.GetFacade(defaultFacade)
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		msg, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		log.Debug().Str("body", string(msg)).Msg("starting to process message")

		if r.URL.Query().Has("facade") {
			facade = application.GetFacade(r.URL.Query().Get("facade"))
		}

		sensorEvent, err := facade(msg)
		if err != nil {
			log.Error().Err(err).Msg("failed to decode sensor event using facade")
		}

		err = a.app.HandleSensorEvent(ctx, sensorEvent)
		if err != nil {
			log.Error().Err(err).Msg("failed to handle message")

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))

			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func (a *api) incomingLWM2MMessageHandler(ctx context.Context) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-lwm2m-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		msg, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		log.Debug().Str("body", string(msg)).Msg("starting to process message")

		pack := senml.Pack{}
		err = json.Unmarshal(msg, &pack)

		if err == nil && len(pack) == 0 {
			err = errors.New("empty senML pack received")
		}

		if err != nil {
			log.Error().Err(err).Msg("failed to decode incoming senML pack")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		deviceID := pack[0].StringValue
		err = a.app.HandleSensorMeasurementList(ctx, deviceID, pack)

		if err != nil {
			log.Error().Err(err).Msg("failed to handle measurement list")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
