package inbox

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	broker "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

//go:gen mockgen -source=message_broker/pkg/inbox/worker.go -destination=message_broker/pkg/inbox/event_consumer_mock.go -package=inbox
type EventConsumer interface {
	// RetrieveMessages returns a slice of messages from the queue or an error
	RetrieveMessages(ctx context.Context) ([]broker.Message, error)
	// ProcessMessages processes the messages the implementor wants to handle
	ProcessMessages(ctx context.Context, messages []broker.Message)
}

type Worker struct {
	EventConsumer
	workerCount int
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewWorker(handler EventConsumer, workerCount int) *Worker {
	return &Worker{EventConsumer: handler, workerCount: workerCount}
}

func (c *Worker) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	msgChan := make(chan []broker.Message, c.workerCount*2)

	for i := 0; i < c.workerCount; i++ {
		c.wg.Go(func() {
			c.doWork(ctx, msgChan)
		})
	}

	c.wg.Go(func() {
		c.poll(ctx, msgChan)
		close(msgChan)
	})
}

func (c *Worker) doWork(ctx context.Context, msgChan chan []broker.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-msgChan:
			if !ok {
				return
			}
			c.ProcessMessages(ctx, batch)
		}
	}
}
func (c *Worker) poll(ctx context.Context, msgChan chan<- []broker.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			messages, err := c.RetrieveMessages(ctx)
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

func (c *Worker) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}
