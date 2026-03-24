package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

const OutboxTableName = "outbox"

type Connector interface {
	*sql.DB | *sql.Tx
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func CreateTable[T Connector](c T) error {

	q := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`
	_, err := c.Exec(q)
	if err != nil {
		return err
	}
	q = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS 
%s (id serial PRIMARY KEY,
event_id uuid NOT NULL UNIQUE,
type varchar NOT NULL ,
correlation_id uuid NOT NULL,
producer varchar NOT NULL,
emitted_at timestamp,
payload jsonb NOT NULL
)`, OutboxTableName)
	_, err = c.Exec(q)
	return err
}

func DropTable[T Connector](c T) error {
	q := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, OutboxTableName)
	_, err := c.Exec(q)
	return err
}

func Insert[T Connector](ctx context.Context, c T, e Event) error {
	//TODO: validate event
	q := fmt.Sprintf("INSERT INTO %s (event_id, type, correlation_id, producer, emitted_at, payload) VALUES ($1, $2, $3, $4, $5, $6)", OutboxTableName)
	_, err := c.ExecContext(ctx, q, e.EventId, e.Type, e.CorrelationId, e.Producer, e.EmittedAt, e.Payload)
	return err
}

func FindAllNewEvents[T Connector](ctx context.Context, c T) ([]Event, error) {
	q := fmt.Sprintf(`SELECT event_id, type, correlation_id, producer, payload FROM %s WHERE emitted_at IS NULL`, OutboxTableName)
	rows, err := c.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]Event, 0)
	for rows.Next() {
		e := Event{}
		err := rows.Scan(&e.EventId, &e.Type, &e.CorrelationId, &e.Producer, &e.Payload)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func MarkAsEmitted[T Connector](ctx context.Context, c T, e []Event) error {
	ids := make([]string, len(e))
	for i := 0; i < len(e); i++ {
		ids[i] = e[i].EventId.String()
	}
	q := fmt.Sprintf(`UPDATE %s SET emitted_at = NOW() WHERE event_id = ANY($1)`, OutboxTableName)
	_, err := c.ExecContext(ctx, q, pq.Array(ids))
	return err
}
