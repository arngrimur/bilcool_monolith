package inbox

// TODO: Create worker to read from queue
// TODO: store messsageId to inbox
// TODO read payload and store in database

// Reviceves
// - booking_ended event
// - user_created event
// - user_deleted event

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	broker "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

// Consumer handles receiving and processing messages from SQS
type Consumer struct {
	handler     *sqs.SqsSubscriber
	workerCount int
	queueUrl    string
	eventsRepo  *postgres.EventRepository
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewConsumer creates a new Consumer instance
func NewConsumer(client *sqs.SqsSubscriber, noWorkers int, db *sql.DB) *Consumer {
	return &Consumer{
		handler:     client,
		workerCount: noWorkers,
		eventsRepo:  postgres.NewEventRepository(db),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	msgChan := make(chan []broker.Message, c.workerCount*2)

	for i := 0; i < c.workerCount; i++ {
		c.wg.Go(func() {
			c.worker(ctx, i, msgChan)
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

// worker processes messages from the channel
func (c *Consumer) worker(ctx context.Context, id int, msgChan chan []broker.Message) {
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

	ok := make([]broker.Message, 0)

	for _, e := range messages {
		err := c.eventsRepo.SaveMessage(processedCtx, e)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to save message")
			continue
		}
		ok = append(ok, e)
	}

	n, err := c.handler.DeleteMessages(processedCtx, ok)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to delete messages")
	}
	log.Info().Int("deleted_messages", n)
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
