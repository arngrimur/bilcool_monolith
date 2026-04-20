package postgresql

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/config"
)

func SetupPostgresDatabase() *sql.DB {
	psqlDb, err := sql.Open("postgres", config.DatabaseUrl())
	if err != nil {
		log.Fatal().Err(err).Msg("error opening database connection")
	}
	maxTries := 10
	for i := 1; i <= maxTries; i++ {
		if err := psqlDb.Ping(); err != nil {
			time.Sleep(1 * time.Second)
			log.Err(err).Msgf("error pinging database, attempt: %d", i)
		} else {
			log.Info().Msg("database connection successful")
			psqlDb.SetMaxOpenConns(5)
			psqlDb.SetMaxIdleConns(5)
			psqlDb.SetConnMaxLifetime(4 * time.Minute)
			psqlDb.SetConnMaxIdleTime(2 * time.Minute)
			return psqlDb
		}
	}
	log.Fatal().Msgf("Error pinging database, gave up after %d attempts", maxTries)
	return nil
}
