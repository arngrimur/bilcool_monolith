package event_dispatcher

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"

	soutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type SnsDispatcher[T postgres.Connector] struct {
	ctx context.Context
	*sns.SnsPublisher
	connector T
}

func NewSnsDispatcher[T postgres.Connector](ctx context.Context, connector T, awsConfig aws.Config) (*SnsDispatcher[T], error) {
	publisher, err := sns.NewPublisher(ctx, awsConfig)
	if err != nil {
		return nil, err
	}
	return &SnsDispatcher[T]{
		ctx:          ctx,
		SnsPublisher: publisher,
		connector:    connector,
	}, nil
}

// Execute publish Sns Notifications that subscribers can retrieve from their personal Sqs Queue
func (s SnsDispatcher[T]) Execute(ctx context.Context, table soutbox.Table) error {
	events, err := postgres.FindAllNewEvents(ctx, s.connector)
	if err != nil {
		return err
	}
	// TODO: Divide into batches of 10
	for _, e := range events {
		e.EmittedAt = new(time.Now())
	}
	messages, err := s.SendBatchMessages(ctx, events, "bookings")
	if err != nil {
		return err
	}

	successfulEvents := make([]postgres.Event, len(messages.Successful))
	for i, m := range messages.Successful {
		uid, err := uuid.Parse(*m.MessageId)
		if err != nil {
			continue
		}
		successfulEvents[i] = postgres.Event{
			EventId: uid,
		}
	}
	return postgres.MarkAsEmitted(ctx, s.connector, events)
}
