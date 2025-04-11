package facades

import (
	"strings"

	"github.com/diwise/iot-agent/internal/pkg/application/facades/chirpstack"
	"github.com/diwise/iot-agent/internal/pkg/application/facades/netmore"
	"github.com/diwise/iot-agent/internal/pkg/application/facades/servanet"
	
	. "github.com/diwise/iot-agent/internal/pkg/application/types"
)

type UplinkEventFunc func([]byte) (SensorEvent, error)
type StatusEventFunc func([]byte) (SensorEvent, error)
type ErrorEventFunc func([]byte) (SensorEvent, error)

func New(as string) UplinkEventFunc {
	switch strings.ToLower(as) {
	case "chirpstack":
		return chirpstack.HandleUplinkEvent
	case "netmore":
		return netmore.HandleUplinkEvent
	case "servanet":
		return servanet.HandleUplinkEvent
	default:
		return chirpstack.HandleUplinkEvent
	}
}
