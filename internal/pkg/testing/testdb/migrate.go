package testdb

import (
	"bytes"
	"database/sql"
	"io"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/amacneil/dbmate/pkg/dbmate"
	pgdriver "github.com/amacneil/dbmate/pkg/driver/postgres"
)

const postgresScheme = "postgres"

// Migrater migrates databases.
type Migrater interface {
	// Migrate migrates db with database URL u.
	Migrate(db *sql.DB, u *url.URL) error
}

// DBMate supports migrating databases using dbmate.
type DBMate struct {
	// Strategy for finding the project root.
	ProjectRoot ProjectRoot
	Log         io.Writer
	// Allows for overriding the migrations table name. Uses default defined by
	// dbmate if left empty.
	MigrationsTableName string
}

// NewDBMate returns a new DBMate instance with the GoModule project root
// strategy.
//
// By default logs are discarded, unless -v flag is passed to go test.
func NewDBMate(t *testing.T, optionsFunc ...OptionsFunc) *DBMate {
	log := io.Discard
	if testing.Verbose() {
		log = NewWriteLogger(t, &bytes.Buffer{})
	}
	d := &DBMate{ProjectRoot: GoModule, Log: log}
	for _, f := range optionsFunc {
		f(d)
	}
	return d
}

// Migrate implements the Migrater interface. Looks for migration files in the
// migrations/ directory in the project's root directory defined by ProjectRoot.
func (m *DBMate) Migrate(db *sql.DB, u *url.URL) error {
	projectRoot, err := m.ProjectRoot.getDir()
	if err != nil {
		return err
	}
	dbmate.RegisterDriver(pgdriver.NewDriver, postgresScheme)
	dbm := dbmate.New(u)
	dbm.MigrationsDir = filepath.Join(projectRoot, "migrations")
	if m.MigrationsTableName != "" {
		dbm.MigrationsTableName = m.MigrationsTableName
	}
	dbm.AutoDumpSchema = false
	dbm.Log = m.Log
	return dbm.Migrate()
}
