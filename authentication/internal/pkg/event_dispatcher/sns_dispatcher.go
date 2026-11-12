package event_dispatcher

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"

	soutbox_domain "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/messaging"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

// Dispatcher publishes the new outbox events through a messaging.Publisher and marks the accepted ones as emitted.
type Dispatcher[T postgres.Connector] struct {
	messaging.Publisher
	connector T
}

func NewDispatcher[T postgres.Connector](connector T, publisher messaging.Publisher) *Dispatcher[T] {
	return &Dispatcher[T]{
		Publisher: publisher,
		connector: connector,
	}
}

func (s Dispatcher[T]) Execute(ctx context.Context, table soutbox_domain.Table) error {
	events, err := postgres.FindAllNewEvents(ctx, s.connector)
	if err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			log.Ctx(ctx).Info().Msg("no new events")
			return nil
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to find new events")
		return err
	}

	log.Ctx(ctx).Info().Int("no new events", len(events)).Msg("found new events")
	for i := range events {
		events[i].EmittedAt = new(time.Now())
	}
	messages, err := s.PublishBatch(ctx, sns.TopicUsers, events)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to send batch messages")
		return err
	}
	successfulEvents := make([]postgres.Event, 0, len(messages.Successful))
	log.Ctx(ctx).Info().Int("successful messages", len(messages.Successful)).Msg("sent messages")
	for _, id := range messages.Successful {
		successfulEvents = append(successfulEvents, postgres.Event{EventId: id})
	}
	return postgres.MarkAsEmitted(ctx, s.connector, successfulEvents)
}
