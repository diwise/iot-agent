package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application"
	"github.com/diwise/iot-agent/internal/pkg/application/iotagent"
	"github.com/diwise/iot-agent/internal/pkg/presentation/api/auth"
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

func New(ctx context.Context, r chi.Router, facade, forwardingEndpoint string, app iotagent.App, policies io.Reader) API {
	return newAPI(ctx, r, facade, forwardingEndpoint, app, policies)
}

func newAPI(ctx context.Context, r chi.Router, facade, forwardingEndpoint string, app iotagent.App, policies io.Reader) *api {
	log := logging.GetFromContext(ctx)

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

	r.Route("/api/v0/measurements", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			authenticator, err := auth.NewAuthenticator(ctx, log, policies)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create api authenticator")
			}
			r.Use(authenticator)
			r.Get("/{id}", a.getMeasurementsHandler(ctx))
		})
	})

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

func (a *api) getMeasurementsHandler(ctx context.Context) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		ctx, span := tracer.Start(r.Context(), "retrieve-measurements")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		deviceID, _ := url.QueryUnescape(chi.URLParam(r, "id"))
		if deviceID == "" {
			err = fmt.Errorf("no device id is supplied in query")
			log.Error().Err(err).Msg("bad request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		device, err := a.app.GetDevice(ctx, deviceID)
		if err != nil {
			log.Error().Err(err).Msg("could not get device information")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		allowedTenants := auth.GetAllowedTenantsFromContext(r.Context())

		if !contains(allowedTenants, device.Tenant()) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		temprel := strings.ToLower(r.URL.Query().Get("temprel")) // before, after, between
		startTime := r.URL.Query().Get("time")                   // 2017-12-13T14:20:00Z
		endTime := r.URL.Query().Get("endTime")                  // 2018-01-13T14:20:00Z

		t := time.Unix(0, 0)
		et := time.Now().UTC()

		if temprel == "before" || temprel == "after" || temprel == "between" {
			t, err = time.Parse(time.RFC3339, startTime)
			if err != nil {
				log.Error().Err(err).Msg("invalid time parameter")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if temprel == "between" {
				et, err = time.Parse(time.RFC3339, endTime)
				if err != nil {
					log.Error().Err(err).Msg("invalid endTime parameter")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}
		} else if len(temprel) > 0 {
			log.Error().Msgf("invalid temprel parameter - %s != before || after || between", temprel)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		l := 1000

		if lastN := r.URL.Query().Get("lastn"); len(lastN) > 0 {
			l, err = strconv.Atoi(lastN)
			if err != nil {
				log.Error().Err(err).Msg("invalid lastN parameter")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		measurements, err := a.app.GetMeasurements(ctx, deviceID, temprel, t, et, l)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		sortOrder := strings.ToLower(r.URL.Query().Get("sort"))

		sort.SliceStable(measurements, func(i, j int) bool {
			if sortOrder == "desc" {
				return measurements[i].Timestamp.After(measurements[j].Timestamp)
			} else {
				return measurements[i].Timestamp.Before(measurements[j].Timestamp)
			}
		})

		b, _ := json.MarshalIndent(measurements, "  ", "  ")

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func contains(arr []string, s string) bool {
	for _, str := range arr {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}
