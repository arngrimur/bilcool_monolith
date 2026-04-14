package event_dispatcher

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	soutbox_domain "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type SnsDispatcher[T postgres.Connector] struct {
	sns.Publisher
	connector T
}

func NewSnsDispatcher[T postgres.Connector](ctx context.Context, connector T, awsConfig aws.Config) (*SnsDispatcher[T], error) {
	publisher, err := sns.NewPublisher(ctx, awsConfig)
	if err != nil {
		return nil, err
	}
	return &SnsDispatcher[T]{
		Publisher: publisher,
		connector: connector,
	}, nil
}

func (s SnsDispatcher[T]) Execute(ctx context.Context, table soutbox_domain.Table) error {
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
	for _, e := range events {
		e.EmittedAt = new(time.Now())
	}
	messages, err := s.SendBatchMessages(ctx, events, sns.TopicUsers)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to send batch messages")
		return err
	}
	successfulEvents := make([]postgres.Event, 0, len(messages.Successful))
	log.Ctx(ctx).Info().Int("successful messages", len(messages.Successful)).Msg("sent messages")
	for _, m := range messages.Successful {
		uid, err := uuid.Parse(*m.Id)
		if err != nil {
			continue
		}
		successfulEvents = append(successfulEvents, postgres.Event{EventId: uid})
	}
	return postgres.MarkAsEmitted(ctx, s.connector, successfulEvents)
}
