package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/pprof"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
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

type PayloadStorer interface {
	Save(ctx context.Context, se application.SensorEvent) error
}

func New(ctx context.Context, r chi.Router, facade, forwardingEndpoint string, app iotagent.App, storer PayloadStorer) (API, error) {
	return newAPI(ctx, r, facade, forwardingEndpoint, app, storer)
}

func newAPI(ctx context.Context, r chi.Router, facade, forwardingEndpoint string, app iotagent.App, storer PayloadStorer) (*api, error) {
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

	r.Post("/api/v0/messages", a.incomingMessageHandler(ctx, facade, storer))
	r.Post("/api/v0/messages/lwm2m", a.incomingLWM2MMessageHandler(ctx))
	r.Post("/api/v0/messages/schneider", a.incomingSchneiderMessageHandler(ctx))

	r.Get("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
	r.Get("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)

	return a, nil
}

func (a *api) Router() chi.Router {
	return a.r
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) incomingMessageHandler(ctx context.Context, defaultFacade string, storer PayloadStorer) http.HandlerFunc {
	facade := application.GetFacade(defaultFacade)
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		msg, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		if r.URL.Query().Has("facade") {
			facade = application.GetFacade(r.URL.Query().Get("facade"))
		}

		sensorEvent, err := facade(msg)
		if err != nil {
			log.Error("failed to decode sensor event using facade", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		err = storer.Save(ctx, sensorEvent)
		if err != nil {
			log.Warn("could not store sensor event", "err", err.Error())
		}

		err = a.app.HandleSensorEvent(ctx, sensorEvent)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
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
		err = a.app.HandleSensorMeasurementList(ctx, deviceID, pack)

		if err != nil {
			log.Error("failed to handle measurement list", "err", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
