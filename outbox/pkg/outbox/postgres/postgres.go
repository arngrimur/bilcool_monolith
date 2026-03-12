package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

const OutboxTableName = "outbox"

type Connector interface {
	*sql.DB | *sql.Tx
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
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
payload bytea NOT NULL
)`, OutboxTableName)
	_, err = c.Exec(q)
	return err
}

func DropTable[T Connector](c T) error {
	q := `DROP TABLE IF EXISTS outbox`
	_, err := c.Exec(q)
	return err
}

func Insert[T Connector](ctx context.Context, c T, e Event) error {
	//TODO: validate event
	q := `INSERT INTO outbox (event_id, type, correlation_id, producer, emitted_at, payload) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := c.ExecContext(ctx, q, e.EventId, e.Type, e.CorrelationId, e.Producer, e.EmittedAt, e.Payload)
	return err
}
