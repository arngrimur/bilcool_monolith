package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	pgdriver "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	"github.com/lib/pq"
)

const OutboxTableName = "outbox"
const migrationsTableName = "outbox_schema_migrations"

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Connector interface {
	*sql.DB | *sql.Tx
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func CreateTable(u *url.URL) error {
	dbmate.RegisterDriver(pgdriver.NewDriver, "postgres")
	dbm := dbmate.New(u)
	dbm.AutoDumpSchema = false
	dbm.Log = io.Discard
	dbm.MigrationsTableName = migrationsTableName

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "outbox_migrations")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tmpDir, entry.Name()), data, 0644); err != nil {
			return err
		}
	}

	dbm.MigrationsDir = []string{tmpDir}
	return dbm.Migrate()
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
