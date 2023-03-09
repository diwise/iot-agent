package messageprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/diwise/iot-agent/internal/pkg/application/conversion"
	"github.com/diwise/iot-agent/internal/pkg/application/decoder/payload"
	dmc "github.com/diwise/iot-device-mgmt/pkg/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/farshidtz/senml/v2"
)

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, p payload.Payload, device dmc.Device) ([]senml.Pack, error)
}

type msgProcessor struct {
	conReg conversion.ConverterRegistry
}

func NewMessageReceivedProcessor(conReg conversion.ConverterRegistry) MessageProcessor {
	return &msgProcessor{
		conReg: conReg,
	}
}

func (mp *msgProcessor) ProcessMessage(ctx context.Context, p payload.Payload, device dmc.Device) ([]senml.Pack, error) {
	log := logging.GetFromContext(ctx)

	if p.Status().Code == payload.PayloadError {
		log.Info().Msg("ignoring payload due to device error")
		return []senml.Pack{}, nil
	}

	messageConverters := mp.conReg.DesignateConverters(ctx, device.Types())
	nrofConverters := len(messageConverters)
	if nrofConverters == 0 {
		return nil, errors.New("no matching converters for device")
	}

	conversionErrors := make([]error, 0, nrofConverters)
	conversionResults := make([]senml.Pack, 0, nrofConverters)

	for _, convert := range messageConverters {
		err := convert(ctx, device.ID(), p, func(sp senml.Pack) error {
			if err := sp.Validate(); err != nil {
				return fmt.Errorf("invalid senML package: %w", err)
			}

			conversionResults = append(conversionResults, sp)

			return nil
		})

		if err != nil {
			conversionErrors = append(conversionErrors, err)
		}
	}

	if len(conversionErrors) > 0 {
		log.Warn().Msgf("%d out of %d converters failed: %v", len(conversionErrors), len(messageConverters), conversionErrors)
	}

	return conversionResults, nil
}
