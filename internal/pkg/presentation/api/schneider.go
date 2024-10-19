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

	"github.com/diwise/iot-agent/pkg/lwm2m"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/google/uuid"
)

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

		log.Debug("starting to process message", "body", string(msg))

		dataList := []SchneiderPayload{}

		err = json.Unmarshal(msg, &dataList)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		for _, d := range dataList {
			id := deterministicGUID(d.ID)

			d.Name, err = trimNameAndReplaceChars(replacer, d.Name)
			if err != nil {
				log.Error("failed to trim name", "err", err.Error())

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			log.Debug(strings.Join(getDeviceConfigString(id, d), ";"))

			value, err := strconv.ParseFloat(d.Value, 64)
			if err != nil {
				log.Error("failed to parse value", "err", err.Error())

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			var object lwm2m.Lwm2mObject

			switch d.Unit {
			case "°C":
				object = lwm2m.NewTemperature(id, value, time.Now().UTC())
			case "Wh":
				object = lwm2m.NewEnergy(id, value, time.Now().UTC())
			case "W":
				object = lwm2m.NewPower(id, value, time.Now().UTC())
			case "%":
				bat := int(value)
				obj := lwm2m.NewDevice(id, time.Now().UTC())
				obj.BatteryLevel = &bat
				object = obj
			default:
				log.Warn("unknown unit", "unit", d.Unit)
			}

			if object == nil {
				continue
			}

			b, _ := json.Marshal(object)

			resp, err := http.Post(fmt.Sprintf("%s/lwm2m", a.forwardingEndpoint), "application/json", bytes.NewBuffer(b))
			if err != nil {
				log.Error("failed to post senml pack", "err", err.Error())

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			defer func() {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusCreated {
				log.Error(fmt.Sprintf("request failed, expected status code %d but got status code %d ", http.StatusCreated, resp.StatusCode))

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("request failed"))
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func getDeviceConfigString(id string, v SchneiderPayload) []string {
	t := make([]string, 0)

	if v.Unit == "°C" {
		t = append(t, lwm2m.Temperature{}.ObjectURN())
	}
	if v.Unit == "Wh" {
		t = append(t, lwm2m.Energy{}.ObjectURN())
	}
	if v.Unit == "W" {
		t = append(t, lwm2m.Power{}.ObjectURN())
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

type SchneiderPayload struct {
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
