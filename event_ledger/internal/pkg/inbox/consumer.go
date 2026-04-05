package inbox

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	broker "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type EventStore interface {
	SaveEvent(ctx context.Context, item domain.EventItem) error
}

type Consumer struct {
	handler     *sqs.SqsSubscriber
	workerCount int
	store       EventStore
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewConsumer(client *sqs.SqsSubscriber, noWorkers int, store EventStore) *Consumer {
	return &Consumer{
		handler:     client,
		workerCount: noWorkers,
		store:       store,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	msgChan := make(chan []broker.Message, c.workerCount*2)

	for i := 0; i < c.workerCount; i++ {
		c.wg.Go(func() {
			c.worker(ctx, msgChan)
		})
	}

	go func() {
		c.poll(ctx, msgChan)
		close(msgChan)
	}()
}

func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

func (c *Consumer) worker(ctx context.Context, msgChan chan []broker.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-msgChan:
			if !ok {
				return
			}
			c.processMessages(ctx, batch)
		}
	}
}

func (c *Consumer) processMessages(ctx context.Context, messages []broker.Message) {
	processedCtx, cancel := context.WithTimeout(ctx, time.Duration(c.handler.VisibilityTimeout()-5)*time.Second)
	defer cancel()

	ok := make([]broker.Message, 0, len(messages))

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

func (c *Consumer) poll(ctx context.Context, msgChan chan<- []broker.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			messages, err := c.handler.RetrieveMessages(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error().Err(err).Msg("error retrieving messages")
				continue
			}
			if len(messages) == 0 {
				continue
			}
			select {
			case msgChan <- messages:
			case <-ctx.Done():
				return
			}
		}
	}
}

func messageToEventItem(m broker.Message) domain.EventItem {
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
