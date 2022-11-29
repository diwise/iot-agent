package events

import (
	"context"
	"errors"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/rs/zerolog"

	"github.com/diwise/messaging-golang/pkg/messaging"
)

//go:generate moq -rm -out eventsender_mock.go . EventSender

type EventSender interface {
	Start() error
	Send(ctx context.Context, m messaging.CommandMessage) error
	Publish(ctx context.Context, m messaging.TopicMessage) error
	Stop() error
}

type eventSender struct {
	rmqConfig    messaging.Config
	rmqMessenger messaging.MsgContext
	started      bool
}

func NewEventSender(serviceName string, logger zerolog.Logger) EventSender {
	sender := &eventSender{
		rmqConfig: messaging.LoadConfiguration(serviceName, logger),
	}

	return sender
}

func (e *eventSender) Send(ctx context.Context, m messaging.CommandMessage) error {
	log := logging.GetFromContext(ctx)

	if !e.started {
		err := errors.New("attempt to send before start")
		log.Error().Err(err).Msg("send failed")
		return err
	}

	log.Info().Msg("sending command to iot-core queue")
	return e.rmqMessenger.SendCommandTo(ctx, m, "iot-core")
}

func (e *eventSender) Publish(ctx context.Context, m messaging.TopicMessage) error {
	log := logging.GetFromContext(ctx)

	if !e.started {
		err := errors.New("attempt to publish before start")
		log.Error().Err(err).Msg("publish failed")
		return err
	}

	log.Info().Str("topic", m.TopicName()).Msg("publishing event")

	return e.rmqMessenger.PublishOnTopic(ctx, m)
}

func (e *eventSender) Start() error {
	var err error
	e.rmqMessenger, err = messaging.Initialize(e.rmqConfig)
	if err == nil {
		e.started = true
	}
	return err
}

func (e *eventSender) Stop() error {
	e.rmqMessenger.Close()
	return nil
}
