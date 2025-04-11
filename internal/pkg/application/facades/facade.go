package facades

import (
	"strings"

	"github.com/diwise/iot-agent/internal/pkg/application/facades/chirpstack"
	"github.com/diwise/iot-agent/internal/pkg/application/facades/netmore"
	"github.com/diwise/iot-agent/internal/pkg/application/facades/servanet"

	. "github.com/diwise/iot-agent/internal/pkg/application/types"
)

type EventFunc func(string, []byte) (Event, error)

func New(as string) EventFunc {
	switch strings.ToLower(as) {
	case "chirpstack":
		return chirpstack.HandleEvent
	case "netmore":
		return netmore.HandleEvent
	case "servanet":
		return servanet.HandleEvent
	default:
		return chirpstack.HandleEvent
	}
}
