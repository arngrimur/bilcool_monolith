package testdb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type SuiteDbIntegration struct {
	Db                *sql.DB
	PostgresContainer *testcontainers.DockerContainer
	CancelFunc        context.CancelFunc
}

// SetupDatabase sets up a database for testing.
// connUrl is a template for the database connection URL in form "postgres://postgres:postgres@localhost:%s/bookings?sslmode=disable"
// fs is a reference to the migrations files
func SetupDatabase(t *testing.T, connUrl string, fs embed.FS) SuiteDbIntegration {
	t.Helper()
	suiteDb := SuiteDbIntegration{}
	ctx := context.Background()
	ctx, suiteDb.CancelFunc = context.WithCancel(ctx)

	var err error
	suiteDb.PostgresContainer, err = testcontainers.Run(
		ctx, "postgres:18",
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
		testcontainers.WithName("bookings_test_db"),
		testcontainers.WithEnv(map[string]string{"POSTGRES_PASSWORD": "postgres", "POSTGRES_USER": "postgres", "POSTGRES_DB": "bookings"}),
	)
	require.NoError(t, err)
	port, err := suiteDb.PostgresContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	u, err := url.Parse(fmt.Sprintf(connUrl, port.Port()))
	require.NoError(t, err)

	suiteDb.Db, err = sql.Open("postgres", u.String())
	require.NoError(t, err)

	dbMate := NewDBMate(t, WithEmbeddedFs(fs))
	err = dbMate.Migrate(suiteDb.Db, u)
	require.NoError(t, err)
	return suiteDb

}
