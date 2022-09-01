package decoder

import (
	"context"
	"encoding/json"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Payload struct {
	DevEUI            string        `json:"devEUI"`
	DeviceName        string        `json:"deviceName,omitempty"`
	FPort             string        `json:"fPort,omitempty"`
	SpreadingFactor   string        `json:"spreadingFactor,omitempty"`
	Latitude          float64       `json:"latitude,omitempty"`
	Longitude         float64       `json:"longitude,omitempty"`
	Rssi              string        `json:"rssi,omitempty"`
	Snr               string        `json:"snr,omitempty"`
	GatewayIdentifier string        `json:"gatewayIdentifier,omitempty"`
	SensorType        string        `json:"sensorType,omitempty"`
	Timestamp         string        `json:"timestamp,omitempty"`
	Type              string        `json:"type,omitempty"`
	Error             string        `json:"error,omitempty"`
	Measurements      []interface{} `json:"measurements"`
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
