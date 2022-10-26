package api

import (
	"io"
	"net/http"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("iot-agent/api")

type API interface {
	Start(port string) error
	health(w http.ResponseWriter, r *http.Request)
}

type api struct {
	log zerolog.Logger
	r   chi.Router
	app iotagent.IoTAgent
}

func NewApi(logger zerolog.Logger, r chi.Router, app iotagent.IoTAgent) API {
	a := newAPI(logger, r, app)

	return a
}

func newAPI(logger zerolog.Logger, r chi.Router, app iotagent.IoTAgent) *api {
	a := &api{
		log: logger,
		r:   r,
		app: app,
	}

	facade := env.GetVariableOrDefault(logger, "APPSERVER_FACADE", "chirpstack")

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	serviceName := "iot-agent"

	r.Use(otelchi.Middleware(serviceName, otelchi.WithChiRoutes(r)))

	r.Get("/health", a.health)
	r.Post("/api/v0/messages", a.incomingMessageHandler(facade))

	return a
}

func (a *api) Start(port string) error {
	a.log.Info().Str("port", port).Msg("starting to listen for connections")

	return http.ListenAndServe(":"+port, a.r)
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) incomingMessageHandler(defaultFacade string) http.HandlerFunc {
	facade := application.GetFacade(defaultFacade)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, a.log, ctx)

		msg, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		log.Debug().Msg("starting to process message")

		if r.URL.Query().Has("facade") {
			facade = application.GetFacade(r.URL.Query().Get("facade"))
		}

		err = a.app.MessageReceivedFn(ctx, msg, facade)
		if err != nil {
			log.Error().Err(err).Msg("failed to handle message")
			log.Debug().Msgf("body: \n%s", msg)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))

			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
