package poller

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/messaging"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type Poller struct {
	db        *sql.DB
	publisher messaging.Publisher
	topic     string
}

func New(db *sql.DB, publisher messaging.Publisher, topic string) *Poller {
	return &Poller{db: db, publisher: publisher, topic: topic}
}

// Poll fetches unprocessed outbox records, publishes them through the publisher, and marks
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

	now := time.Now().UTC()
	for i := range events {
		events[i].EmittedAt = &now
	}

	result, err := p.publisher.PublishBatch(ctx, p.topic, events)
	if err != nil {
		return err
	}
	for _, failure := range result.Failed {
		log.Ctx(ctx).Warn().Stringer("event_id", failure.EventId).Str("reason", failure.Reason).Msg("event was not published")
	}

	successful := make([]postgres.Event, 0, len(result.Successful))
	for _, id := range result.Successful {
		successful = append(successful, postgres.Event{EventId: id, EmittedAt: &now})
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
