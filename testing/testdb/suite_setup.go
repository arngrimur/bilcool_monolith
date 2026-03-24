package testdb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type SuiteDbIntegration struct {
	Db                *sql.DB
	PostgresContainer *testcontainers.DockerContainer
	CancelFunc        context.CancelFunc
	Ctx               context.Context
	ConnString        *url.URL
}

func (s *SuiteDbIntegration) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.Db.Exec(query, args...)
}

func (s *SuiteDbIntegration) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.Db.ExecContext(ctx, query, args...)
}

func (s *SuiteDbIntegration) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.Db.QueryContext(ctx, query, args...)
}

// SetupDatabase sets up a database for testing.
// connUrl is a template for the database connection URL in form "postgres://postgres:postgres@localhost:%s/xyz?sslmode=disable"
// fs is a reference to the migrations files
// dbName is the name of the database
func SetupDatabase(t *testing.T, fs embed.FS, dbName string) SuiteDbIntegration {
	t.Helper()
	const connUrl = "postgres://postgres:postgres@localhost:%s/%s?sslmode=disable"
	suiteDb := SuiteDbIntegration{}
	suiteDb.Ctx, suiteDb.CancelFunc = context.WithCancel(context.Background())

	var err error

	withDummyPort, err := url.Parse(fmt.Sprintf(connUrl, "1", dbName))
	require.NoError(t, err)

	suiteDb.PostgresContainer, err = testcontainers.Run(
		suiteDb.Ctx, "postgres:18",
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
		testcontainers.WithName(dbName),
		testcontainers.WithEnv(map[string]string{"POSTGRES_PASSWORD": "postgres", "POSTGRES_USER": "postgres", "POSTGRES_DB": withDummyPort.Path[1:]}),
		//"-c wal_level=logical -c max_wal_senders=5 -c max_replication_slots=5"
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.Cmd = []string{"-c", "wal_level=logical", "-c", "max_wal_senders=5", "-c", "max_replication_slots=5"}
		}),
	)
	require.NoError(t, err)

	port, err := suiteDb.PostgresContainer.MappedPort(suiteDb.Ctx, "5432/tcp")
	require.NoError(t, err)
	u, err := url.Parse(fmt.Sprintf(connUrl, port.Port(), dbName))
	require.NoError(t, err)
	suiteDb.Db, err = sql.Open("postgres", u.String())
	require.NoError(t, err)
	suiteDb.ConnString = u

	dbMate := NewDBMate(t, WithEmbeddedFs(fs))
	err = dbMate.Migrate(suiteDb.Db, u)
	if err != nil && err.Error() == "no migration files found" {
		return suiteDb
	}
	require.NoError(t, err)
	return suiteDb

}
func (s *SuiteDbIntegration) TearDown(t *testing.T) {
	s.CancelFunc()
	_ = s.Db.Close()
	testcontainers.CleanupContainer(t, s.PostgresContainer)
}
