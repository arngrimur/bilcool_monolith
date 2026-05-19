package postgresql

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
)

func SetupPostgresDatabase() *sql.DB {
	dbURL := config.DatabaseUrl()
	if u, err := url.Parse(dbURL); err == nil {
		log.Info().Str("host", u.Host).Str("db", u.Path).Msg("connecting to database")
	}

	psqlDb, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("error opening database connection")
	}

	const maxTries = 10
	var lastErr error
	for i := 1; i <= maxTries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		lastErr = psqlDb.PingContext(ctx)
		cancel()
		if lastErr == nil {
			log.Info().Msg("database connection successful")
			if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
				psqlDb.SetMaxOpenConns(2)
				psqlDb.SetMaxIdleConns(0)
				psqlDb.SetConnMaxLifetime(30 * time.Second)
				psqlDb.SetConnMaxIdleTime(0)
			} else {
				psqlDb.SetMaxOpenConns(5)
				psqlDb.SetMaxIdleConns(5)
				psqlDb.SetConnMaxLifetime(4 * time.Minute)
				psqlDb.SetConnMaxIdleTime(2 * time.Minute)
			}
			return psqlDb
		}
		log.Error().Err(lastErr).Msgf("error pinging database, attempt %d/%d", i, maxTries)
		time.Sleep(1 * time.Second)
	}
	log.Fatal().Err(lastErr).Msgf("error pinging database, gave up after %d attempts", maxTries)
	return nil
}
