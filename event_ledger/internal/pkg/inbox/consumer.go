package inbox

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type EventStore interface {
	SaveEvent(ctx context.Context, item domain.EventItem) error
}

type Consumer struct {
	handler *sqs.SqsSubscriber
	store   EventStore
}

func NewConsumer(client *sqs.SqsSubscriber, store EventStore) *Consumer {
	return &Consumer{
		handler: client,
		store:   store,
	}
}

func (c *Consumer) ProcessMessages(ctx context.Context, messages []brokerpostgres.Message) {
	processedCtx, cancel := context.WithTimeout(ctx, time.Duration(c.handler.VisibilityTimeout()-5)*time.Second)
	defer cancel()

	ok := make([]brokerpostgres.Message, 0, len(messages))

	for _, m := range messages {
		item := messageToEventItem(m)
		if err := c.store.SaveEvent(processedCtx, item); err != nil {
			log.Ctx(ctx).Err(err).Str("event_id", item.EventId).Msg("failed to save event")
			continue
		}
		ok = append(ok, m)
	}

	n, err := c.handler.DeleteMessages(processedCtx, ok)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to delete messages")
	}
	log.Info().Int("deleted_messages", n).Send()
}

func (c *Consumer) RetrieveMessages(ctx context.Context) ([]brokerpostgres.Message, error) {
	return c.handler.RetrieveMessages(ctx)
}

func messageToEventItem(m brokerpostgres.Message) domain.EventItem {
	emittedAt := ""
	if m.Message.EmittedAt != nil {
		emittedAt = m.Message.EmittedAt.UTC().Format(time.RFC3339Nano)
	}
	return domain.EventItem{
		EventId:       m.Message.EventId.String(),
		EventType:     m.Message.Type,
		CorrelationId: m.Message.CorrelationId.String(),
		Producer:      m.Message.Producer,
		EmittedAt:     emittedAt,
		Payload:       string(m.Message.Payload),
		ReceivedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
}
