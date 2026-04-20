package poller

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

//go:generate mockgen -source=poller.go -destination=poller_mock.go -package=poller

type Poller struct {
	db        *sql.DB
	publisher sns.Publisher
	topic     string
}

func New(db *sql.DB, publisher sns.Publisher, topic string) *Poller {
	return &Poller{db: db, publisher: publisher, topic: topic}
}

// Poll fetches unprocessed outbox records, publishes them to SNS, and marks
// them emitted — all within a single transaction using SKIP LOCKED so
// concurrent Lambda invocations never process the same record twice.
func (p *Poller) Poll(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryContext(ctx,
		`SELECT event_id, type, correlation_id, producer, payload
		 FROM outbox WHERE emitted_at IS NULL
		 LIMIT 100 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return err
	}

	events := make([]postgres.Event, 0)
	for rows.Next() {
		e := postgres.Event{}
		if err := rows.Scan(&e.EventId, &e.Type, &e.CorrelationId, &e.Producer, &e.Payload); err != nil {
			rows.Close()
			return err
		}
		events = append(events, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	results, err := p.publisher.SendBatchMessages(ctx, events, p.topic)
	if err != nil {
		return err
	}

	successful := make([]postgres.Event, 0, len(results.Successful))
	for _, m := range results.Successful {
		uid, err := uuid.Parse(*m.Id)
		if err != nil {
			log.Ctx(ctx).Warn().Str("id", *m.Id).Msg("failed to parse successful message id")
			continue
		}
		successful = append(successful, postgres.Event{EventId: uid})
	}

	if len(successful) == 0 {
		return nil
	}

	if err := postgres.MarkAsEmitted(ctx, tx, successful); err != nil {
		return err
	}
	return tx.Commit()
}

// RunLoop calls Poll on every interval tick until ctx is cancelled.
// Used in long-running deployments when OUTBOX_MODE=polling.
func (p *Poller) RunLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.Poll(ctx); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("outbox poll failed")
			}
		}
	}
}
