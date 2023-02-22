package api

import (
	"context"
	"io"
	"net/http"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
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
	r   chi.Router
	app iotagent.IoTAgent
}

func New(ctx context.Context, r chi.Router, facade string, app iotagent.IoTAgent) API {
	return newAPI(ctx, r, facade, app)
}

func newAPI(ctx context.Context, r chi.Router, facade string, app iotagent.IoTAgent) *api {

	a := &api{
		r:   r,
		app: app,
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

		err = a.app.MessageReceivedFn(ctx, msg, facade)
		if err != nil {
			log.Error().Err(err).Msg("failed to handle message")

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))

			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
