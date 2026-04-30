package main

import (
	"context"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	pgdriver "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/migrations"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("authentication-migrate"))
	log.Ctx(ctx).Info().Msg("starting authentication migrate lambda")

	lambda.Start(func(ctx context.Context) error {
		u, err := url.Parse(os.Getenv("DATABASE_URL"))
		if err != nil {
			return err
		}

		dbmate.RegisterDriver(pgdriver.NewDriver, "postgres")
		dbm := dbmate.New(u)
		dbm.AutoDumpSchema = false
		dbm.Log = io.Discard
		dbm.MigrationsTableName = "schema_migrations"

		entries, err := fs.ReadDir(migrations.FS, ".")
		if err != nil {
			return err
		}
		tmpDir, err := os.MkdirTemp("", "authentication_migrations")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := fs.ReadFile(migrations.FS, entry.Name())
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(tmpDir, entry.Name()), data, 0644); err != nil {
				return err
			}
		}

		dbm.MigrationsDir = []string{tmpDir}
		return dbm.Migrate()
	})
}
