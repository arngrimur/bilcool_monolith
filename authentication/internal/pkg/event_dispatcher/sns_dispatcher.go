package event_dispatcher

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"

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
		return err
	}
	for _, e := range events {
		e.EmittedAt = new(time.Now())
	}
	messages, err := s.SendBatchMessages(ctx, events, sns.TopicUsers)
	if err != nil {
		return err
	}
	successfulEvents := make([]postgres.Event, 0, len(messages.Successful))
	for _, m := range messages.Successful {
		uid, err := uuid.Parse(*m.Id)
		if err != nil {
			continue
		}
		successfulEvents = append(successfulEvents, postgres.Event{EventId: uid})
	}
	return postgres.MarkAsEmitted(ctx, s.connector, successfulEvents)
}
