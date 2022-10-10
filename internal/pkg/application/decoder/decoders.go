package decoder

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Payload struct {
	BatteryLevel      int     `json:"batteryLevel"`
	DevEUI            string  `json:"devEUI"`
	DeviceName        string  `json:"deviceName,omitempty"`
	FPort             string  `json:"fPort,omitempty"`
	GatewayIdentifier string  `json:"gatewayIdentifier,omitempty"`
	Latitude          float64 `json:"latitude,omitempty"`
	Longitude         float64 `json:"longitude,omitempty"`
	Measurements      []any   `json:"measurements"`
	Rssi              string  `json:"rssi,omitempty"`
	SensorType        string  `json:"sensorType,omitempty"`
	Snr               string  `json:"snr,omitempty"`
	SpreadingFactor   string  `json:"spreadingFactor,omitempty"`
	Status            Status  `json:"status"`
	Timestamp         string  `json:"timestamp,omitempty"`
	Type              string  `json:"type,omitempty"`
}

const PAYLOAD_ERROR = 100

type Status struct {
	Code     int      `json:"statusCode"`
	Messages []string `json:"statusMessages"`
}

func (p *Payload) SetStatus(code int, messages []string) {
	p.Status = Status{
		Code:     code,
		Messages: messages,
	}
}

func (p Payload) ConvertToStruct(v any) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	return nil
}

func (p *Payload) ValueOf(name string) any {
	for _, m := range p.Measurements {
		t := reflect.TypeOf(m)

		for i := 0; i < t.NumField(); i++ {
			if strings.EqualFold(t.Field(i).Name, name) {
				v := reflect.ValueOf(m)
				return v.Field(i).Interface()
			}
		}
	}

	return nil
}

type MessageDecoderFunc func(context.Context, []byte, func(context.Context, Payload) error) error

func DefaultDecoder(ctx context.Context, msg []byte, fn func(context.Context, Payload) error) error {
	log := logging.GetFromContext(ctx)

	d := struct {
		DevEUI string `json:"devEUI"`
	}{}

	err := json.Unmarshal(msg, &d)
	if err != nil {
		return err
	}

	p := Payload{
		DevEUI: d.DevEUI,
	}

	log.Info().Msgf("default decoder used for devEUI %s", p.DevEUI)

	return fn(ctx, p)
}
