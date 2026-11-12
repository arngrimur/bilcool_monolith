package event_dispatcher

import (
	"context"
	"time"

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

// Execute publishes the new outbox events to the bookings topic, where each subscriber gets them on its own queue
func (s Dispatcher[T]) Execute(ctx context.Context, table soutbox_domain.Table) error {
	events, err := postgres.FindAllNewEvents(ctx, s.connector)
	if err != nil {
		return err
	}
	for i := range events {
		events[i].EmittedAt = new(time.Now())
	}
	messages, err := s.PublishBatch(ctx, sns.TopicBookings, events)
	if err != nil {
		return err
	}

	successfulEvents := make([]postgres.Event, 0, len(messages.Successful))
	for _, id := range messages.Successful {
		successfulEvents = append(successfulEvents, postgres.Event{EventId: id})
	}
	return postgres.MarkAsEmitted(ctx, s.connector, successfulEvents)
}
