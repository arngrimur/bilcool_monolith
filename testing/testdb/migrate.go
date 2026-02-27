package testdb

import (
	"bytes"
	"database/sql"
	"embed"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	pgdriver "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
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
	Fs                  embed.FS
	UseFs               bool
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
	dbmate.RegisterDriver(pgdriver.NewDriver, postgresScheme)
	dbm := dbmate.New(u)
	dbm.AutoDumpSchema = false
	dbm.Log = m.Log
	if !m.UseFs {
		projectRoot, err := m.ProjectRoot.getDir()
		if err != nil {
			return err
		}
		dir, _ := os.Getwd()

		dbm.MigrationsDir = []string{strings.Join([]string{dir, projectRoot, "migrations"}, "/")}
		if m.MigrationsTableName != "" {
			dbm.MigrationsTableName = m.MigrationsTableName
		}
	} else {
		dir, err := fs.ReadDir(m.Fs, ".")
		if err != nil {
			panic(err)
		}
		tempdir, err := os.MkdirTemp("/tmp", "migrations")
		if err != nil {
			panic(err)
		}
		defer func() {
			os.RemoveAll(tempdir)
		}()
		for _, fsData := range dir {
			if fsData.IsDir() {
				os.MkdirAll(fsData.Name(), 0755)
			} else {
				data, err := fs.ReadFile(m.Fs, fsData.Name())
				if err != nil {
					panic(err)
				}
				err = os.WriteFile(tempdir+"/"+fsData.Name(), data, 0644)
				if err != nil {
					panic(err)
				}
			}
		}
		dbm.MigrationsDir = []string{tempdir}
	}
	return dbm.Migrate()
}
