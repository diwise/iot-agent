package api

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/farshidtz/senml/v2"
	"github.com/google/uuid"
)

var devices map[string]Data

func init() {
	devices = make(map[string]Data, 0)
}

func (a *api) incomingSchneiderMessageHandler(ctx context.Context) http.HandlerFunc {
	logger := logging.GetFromContext(ctx)

	replacer := strings.NewReplacer("_", "-", "å", "a", "ä", "a", "ö", "o")

	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		ctx, span := tracer.Start(r.Context(), "incoming-schneider-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, _, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		msg, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		log.Debug().Str("body", string(msg)).Msg("starting to process message")

		dataList := []Data{}

		err = json.Unmarshal(msg, &dataList)
		if err != nil {
			log.Error().Err(err).Msg("failed to handle message")

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		for _, object := range dataList {
			id := deterministicGUID(object.ID)

			object.Name, err = trimNameAndReplaceChars(replacer, object.Name)
			if err != nil {
				log.Error().Err(err).Msg("failed to trim name")

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			log.Debug().Msg(strings.Join(getDeviceConfigString(id, object), ";"))

			devices[id] = object

			value, err := strconv.ParseFloat(object.Value, 64)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse value")

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			basename := ""
			unit := ""

			if object.Unit == "°C" {
				basename = conversion.TemperatureURN
				unit = senml.UnitCelsius
			} else if object.Unit == "Wh" {
				basename = conversion.EnergyURN
				unit = senml.UnitJoule
				value = value * 3600
			} else if object.Unit == "W" {
				basename = conversion.PowerURN
				unit = senml.UnitWatt
			}

			decorators := []conversion.SenMLDecoratorFunc{
				conversion.ValueWithUnit("5700", unit, value),
			}

			pack := conversion.NewSenMLPack(id, basename, time.Now().UTC(), decorators...)
			b, _ := json.Marshal(pack)

			url := a.forwardingEndpoint + "/lwm2m"

			resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
			if err != nil {
				log.Error().Err(err).Msg("failed to post senml pack")

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				log.Error().Msgf("request failed, expected status code %d but got status code %d ", http.StatusCreated, resp.StatusCode)

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("request failed"))
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func getDeviceConfigString(id string, v Data) []string {
	t := make([]string, 0)

	if v.Unit == "°C" {
		t = append(t, conversion.TemperatureURN)
	}
	if v.Unit == "Wh" {
		t = append(t, conversion.EnergyURN)
	}
	if v.Unit == "W" {
		t = append(t, conversion.PowerURN)
	}

	r := []string{
		v.ID,
		id,
		"0",
		"0",
		"",
		strings.Join(t, ","),
		"virtual",
		v.Name,
		v.Description,
		"true",
		"default",
		"3600",
		"Schneider",
	}

	return r
}

func trimNameAndReplaceChars(replacer *strings.Replacer, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	parts := strings.Split(name, "/")
	slices.Reverse(parts)
	for _, n := range parts {
		if strings.ToLower(n) != "value" {
			return replacer.Replace(n), nil
		}
	}

	return replacer.Replace(name), nil
}

type Data struct {
	ID          string `json:"pointID"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Unit        string `json:"unit"`
	Description string `json:"description"`
}

func deterministicGUID(str string) string {
	md5hash := md5.New()
	md5hash.Write([]byte(str))
	md5string := hex.EncodeToString(md5hash.Sum(nil))

	unique, err := uuid.FromBytes([]byte(md5string[0:16]))
	if err != nil {
		return uuid.New().String()
	}

	return unique.String()
}
